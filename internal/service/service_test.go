package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"io"
	"os"
	"testing"
	"vwap-service/internal/vwap"
)

func TestNewService(t *testing.T) {
	type args struct {
		ctx      context.Context
		streamer Streamer
		opts     []Option
	}
	tests := map[string]struct {
		args args
	}{
		"successfully create a new service": {
			args: args{
				ctx:      context.Background(),
				streamer: new(StreamerMock),
				opts:     []Option{WithLogger(nil), WithOutput(os.Stdout), WithMaxDataPts(50)},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := NewService(tt.args.ctx, tt.args.streamer, tt.args.opts...)

			assert.Equal(t, false, got.running.Load())
			assert.Equal(t, make(vwapRecords), got.vwaps)
			assert.Equal(t, os.Stdout, got.output)
			assert.Equal(t, 50, got.maxDataPts)
		})
	}
}

func TestService_AddTradingPairs(t *testing.T) {
	type fields struct {
		vwaps      vwapRecords
		maxDataPts int
	}
	type args struct {
		tradingPairs []string
	}
	tests := map[string]struct {
		name   string
		fields fields
		args   args
		want   vwapRecords
	}{
		"it should add 2 new trading pairs successfully": {
			args: args{
				tradingPairs: []string{"ETH-BTC"},
			},
			fields: fields{
				vwaps: make(vwapRecords),
			},
			want: vwapRecords{
				"ETH-BTC": &vwapRecord{
					VWaper: vwap.New(200),
					Name:   "ETH-BTC",
				},
			},
		},
		"it should not replace an existing trading pair": {
			args: args{
				tradingPairs: []string{"ETH-BTC", "BTC-USD"},
			},
			fields: fields{
				vwaps: vwapRecords{
					"ETH-BTC": &vwapRecord{
						VWaper: vwap.New(50),
						Name:   "ETH-BTC",
					},
				},
			},
			want: vwapRecords{
				"ETH-BTC": &vwapRecord{
					VWaper: vwap.New(50),
					Name:   "ETH-BTC",
				},
				"BTC-USD": &vwapRecord{
					VWaper: vwap.New(200),
					Name:   "BTC-USD",
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s := &Service{
				vwaps: tt.fields.vwaps,
			}
			s.AddTradingPairs(tt.args.tradingPairs...)

			assert.Equal(t, tt.want, s.vwaps)
		})
	}
}

func TestService_Run_assert_start_and_status(t *testing.T) {
	type fields struct {
		ctx        context.Context
		vwaps      vwapRecords
		logger     *zap.Logger
		streamer   Streamer
		maxDataPts int
		output     io.Writer
		stop       chan bool
		running    *atomic.Bool
	}
	type streamerOutputs struct {
		subscribe error
	}

	tests := map[string]struct {
		fields          fields
		streamerOutputs streamerOutputs
		wantErr         assert.ErrorAssertionFunc
	}{
		"it should error when service is already running": {
			fields: fields{
				running: atomic.NewBool(true),
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				assert.EqualError(t, err, "service is already running")
				return true
			},
		},
		"it should error when no trading-pairs are set": {
			fields: fields{
				vwaps:   vwapRecords{},
				running: atomic.NewBool(false),
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				assert.EqualError(t, err, "no trading pairs were provided")
				return true
			},
		},
		"it should error subscription fails": {
			fields: fields{
				vwaps: vwapRecords{
					"ETH-BTC": {},
				},
				running: atomic.NewBool(false),
			},
			streamerOutputs: streamerOutputs{
				subscribe: errors.New("server error"),
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				assert.EqualError(t, err, "subscribe to matches channel: server error")
				return true
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			for _, record := range tt.fields.vwaps {
				vwapMock := new(VWAPMock)
				vwapMock.On("Push", mock.Anything, mock.Anything).Return(nil)
				record.VWaper = vwapMock
			}

			streamerMock := new(StreamerMock)
			streamerMock.On("Subscribe", mock.Anything, mock.Anything).Return(tt.streamerOutputs.subscribe)

			s := &Service{
				ctx:      context.Background(),
				vwaps:    tt.fields.vwaps,
				running:  tt.fields.running,
				streamer: streamerMock,
			}

			err := s.Run()
			tt.wantErr(t, err)
		})
	}
}

func Test_vwapRecord_updateVWAP(t *testing.T) {
	type args struct {
		price  string
		volume string
	}
	tests := map[string]struct {
		args       args
		pushReturn error
		wantErr    assert.ErrorAssertionFunc
	}{
		"it should error parsing price": {
			args: args{
				price:  "not-a-number",
				volume: "0",
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "parse price 'not-a-number'")
				return true
			},
		},
		"it should error parsing size": {
			args: args{
				price:  "0",
				volume: "not-a-number",
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "parse size 'not-a-number'")
				return true
			},
		},
		"it should error when push fails": {
			args: args{
				price:  "0.5555",
				volume: "0.6666",
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				assert.EqualError(t, err, "push trading-pair to VWAP: some error")
				return true
			},
			pushReturn: errors.New("some error"),
		},
		"it should successfully update VWAP record": {
			args: args{
				price:  "0.5555",
				volume: "0.6666",
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return false
			},
			pushReturn: nil,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			vwaper := new(VWAPMock)
			vwaper.On("Push", mock.Anything, mock.Anything).Return(tt.pushReturn)

			v := &vwapRecord{
				VWaper: vwaper,
				Name:   "TP",
			}

			err := v.updateVWAP(tt.args.price, tt.args.volume)
			tt.wantErr(t, err)
		})
	}
}

func Test_vwapRecords_tradingPairs(t *testing.T) {
	tests := map[string]struct {
		v    vwapRecords
		want []string
	}{
		"it should return an empty list": {
			v:    vwapRecords{},
			want: nil,
		},
		"it should return 3 items": {
			v: vwapRecords{
				"BTC-USD": &vwapRecord{
					VWaper: vwap.New(200),
					Name:   "BTC-USD",
				},
				"ETH-BTC": &vwapRecord{
					VWaper: vwap.New(200),
					Name:   "ETH-BTC",
				},
				"ETH-USD": &vwapRecord{
					VWaper: vwap.New(200),
					Name:   "ETH-USD",
				},
			},
			want: []string{"BTC-USD", "ETH-BTC", "ETH-USD"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.v.tradingPairs()
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestService_parseFeedMsg(t *testing.T) {
	type args struct {
		msg []byte
	}
	tests := map[string]struct {
		args    args
		want    *ExchangeMsg
		wantErr assert.ErrorAssertionFunc
	}{
		"it should successfully return a new ExchangeMessage": {
			args: args{
				msg: []byte(`{"type": "match", "product_id": "ETH-BTC", "price": "5.0", "size": "0.333", "side": "buy"}`),
			},
			want: &ExchangeMsg{
				Type:      "match",
				ProductID: "ETH-BTC",
				Size:      "5.0",
				Price:     "0.333",
				Side:      "buy",
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return false
			},
		},
		"it should throw an error for invalid msg": {
			args: args{
				msg: []byte(`wrong message`),
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				assert.EqualError(t, err, "failed to unmarshal message: invalid character 'w' looking for beginning of value")
				return true
			},
		},
		"it should throw an error if message type is 'error'": {
			args: args{
				msg: []byte(`{"type": "error", "message": "no data"}`),
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				assert.EqualError(t, err, "received an error message from the server: no data")
				return true
			},
		},
		"it should return nothing if type is not 'match'": {
			args: args{
				msg: []byte(`{"type": "last_match", "price": "1.0"}`),
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return false
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s := &Service{}
			got, err := s.parseFeedMsg(tt.args.msg)
			if !tt.wantErr(t, err, fmt.Sprintf("parseFeedMsg(%v)", tt.args.msg)) {
				return
			}
			assert.Equalf(t, tt.want, got, "parseFeedMsg(%v)", tt.args.msg)
		})
	}
}
