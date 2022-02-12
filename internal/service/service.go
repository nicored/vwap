package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"vwap-service/internal/crypto-streamer/coinbase"
	"vwap-service/internal/vwap"
)

const (
	_defaultMaxPts = 200
)

type VWapRecord struct {
	VWaper
	Name string
}

func (v VWapRecord) String() string {
	return v.Name + ": " + strconv.FormatFloat(v.Value(), 'f', 6, 64)
}

type VWapRecords map[string]*VWapRecord

func (v VWapRecords) TradingPairs() []string {
	var pairs []string
	for tp := range v {
		pairs = append(pairs, tp)
	}
	return pairs
}

type Service struct {
	mu         sync.Mutex
	ctx        context.Context
	vwaps      VWapRecords
	logger     *zap.Logger
	streamer   Streamer
	maxDataPts int
	output     io.Writer
}

func NewService(ctx context.Context, streamer Streamer, opts ...Option) *Service {
	options := options{
		logger:     zap.NewNop(),
		maxDataPts: _defaultMaxPts,
		output:     os.Stdout,
	}

	for _, o := range opts {
		o.apply(&options)
	}

	return &Service{
		streamer:   streamer,
		ctx:        ctx,
		vwaps:      make(VWapRecords),
		logger:     options.logger,
		maxDataPts: options.maxDataPts,
		output:     options.output,
	}
}

func (s *Service) Run() error {
	if len(s.vwaps) == 0 {
		return errors.New("no trading pairs were provided")
	}

	if err := s.streamer.Subscribe(coinbase.ChannelMatches, s.vwaps.TradingPairs()...); err != nil {
		return fmt.Errorf("subscribe to channel: %w", err)
	}

	feeds, feedsErr := s.streamer.Feeds()
	for {
		select {
		case <-s.ctx.Done():
			return nil
		case fErr := <-feedsErr:
			return fmt.Errorf("feeds: %w", fErr)
		case msg := <-feeds:
			exchMsg := ExchangeMsg{}
			if err := json.Unmarshal(msg, &exchMsg); err != nil {
				s.logger.Error("service run: failed to unmarshal message", zap.String("msg", string(msg)), zap.NamedError("error", err))
				continue
			}

			if exchMsg.Type == coinbase.TypeError {
				s.logger.Error("service run: received an error message from the server", zap.String("msg", string(msg)))
				continue
			}

			if exchMsg.Type != coinbase.TypeMatch {
				continue
			}

			tpvwap, ok := s.vwaps[exchMsg.ProductID]
			if !ok {
				s.logger.Sugar().Errorf("service run: check trading pair: %s is out of scope", exchMsg.ProductID)
				continue
			}

			if err := tpvwap.updateVWAP(exchMsg.Price, exchMsg.Size); err != nil {
				s.logger.Error("failed to calculate VWAP from feed message", zap.NamedError("error", err), zap.String("msg", string(msg)))
				continue
			}

			if _, err := io.WriteString(s.output, tpvwap.String()+"\n"); err != nil {
				s.logger.Error("failed to write VWAP to output target", zap.NamedError("error", err))
			}
		}
	}
}

func (s *Service) AddTradingPairs(tradingPairs ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, tp := range tradingPairs {
		tp = strings.ToUpper(tp)

		if _, ok := s.vwaps[tp]; !ok {
			s.vwaps[tp] = &VWapRecord{
				VWaper: vwap.New(s.maxDataPts),
				Name:   tp,
			}
		}
	}
}

func (s *Service) Close() error {
	return s.streamer.Close()
}

func (v *VWapRecord) updateVWAP(price string, volume string) error {
	fprice, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return fmt.Errorf("parse fprice '%s': %w", price, err)
	}

	fvolume, err := strconv.ParseFloat(volume, 64)
	if err != nil {
		return fmt.Errorf("parse size '%s': %w", volume, err)
	}

	if err := v.Push(fprice, fvolume); err != nil {
		return fmt.Errorf("push trading-pair to VWAP")
	}

	return nil
}
