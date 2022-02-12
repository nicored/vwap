package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/atomic"
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

// Service is a calculattion engine service used to compute VWAP's for given trading-pairs,
// and output them to a target output streamer is a crypto exchange streamer that implements the Streamer interface
// Available options are WithLogger(logger), WithOutput(output = stdout), WithMaxDataPts(max = 200)
type Service struct {
	mu         sync.Mutex
	ctx        context.Context
	vwaps      vwapRecords
	logger     *zap.Logger
	streamer   Streamer
	maxDataPts int
	output     io.Writer
	stop       chan bool
	running    atomic.Bool
}

// NewService creates a new calculation engine service
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
		vwaps:      make(vwapRecords),
		logger:     options.logger,
		maxDataPts: options.maxDataPts,
		output:     options.output,
		stop:       make(chan bool, 1),
	}
}

// AddTradingPairs creates new vwap records for the given trading pairs
// so the service can subscribe and compute their VWAP. Trading pairs must
// be added before the Run method is executed.
func (s *Service) AddTradingPairs(tradingPairs ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, tp := range tradingPairs {
		tp = strings.ToUpper(tp)

		if _, ok := s.vwaps[tp]; !ok {
			s.vwaps[tp] = &vwapRecord{
				VWaper: vwap.New(s.maxDataPts),
				Name:   tp,
			}
		}
	}
}

// Run reads Streamer feeds and computes the VWAP for returned trading pairs
func (s *Service) Run() error {
	if s.running.Load() == true {
		return errors.New("service is already running")
	}
	s.running.Store(true)
	defer s.running.Store(false)

	// we must have trading pairs to run
	if len(s.vwaps) == 0 {
		return errors.New("no trading pairs were provided")
	}

	// subscribing to the streamer's channel for the available trading pairs
	if err := s.streamer.Subscribe(coinbase.ChannelMatches, s.vwaps.tradingPairs()...); err != nil {
		return fmt.Errorf("subscribe to channel: %w", err)
	}

	feeds, feedsErr := s.streamer.Feeds()
	for {
		select {
		case <-s.stop:
			return nil

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

			s.mu.Lock()
			tpvwap, ok := s.vwaps[exchMsg.ProductID]
			if !ok {
				s.logger.Sugar().Errorf("service run: check trading pair: %s is out of scope", exchMsg.ProductID)
				continue
			}

			if err := tpvwap.updateVWAP(exchMsg.Price, exchMsg.Size); err != nil {
				s.logger.Error("failed to calculate VWAP from feed message", zap.NamedError("error", err), zap.String("msg", string(msg)))
				continue
			}

			if _, err := io.WriteString(s.output, tpvwap.string()+"\n"); err != nil {
				s.logger.Error("failed to write VWAP to output target", zap.NamedError("error", err))
			}
			s.mu.Unlock()
		}
	}
}

// Stop stops the execution of the service
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running.Load() == true {
		s.stop <- true
	}
}

type vwapRecord struct {
	VWaper
	Name string
}

func (v *vwapRecord) updateVWAP(price string, volume string) error {
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

func (v vwapRecord) string() string {
	return v.Name + ": " + strconv.FormatFloat(v.Value(), 'f', 6, 64)
}

type vwapRecords map[string]*vwapRecord

func (v vwapRecords) tradingPairs() []string {
	var pairs []string
	for tp := range v {
		pairs = append(pairs, tp)
	}
	return pairs
}
