package service

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"os"
	"testing"
	"vwap-service/internal/crypto-streamer/coinbase"
)

func TestNewService(t *testing.T) {
	type args struct {
		ctx      context.Context
		streamer Streamer
		opts     []Option
	}
	tests := map[string]struct {
		args args
		want *Service
	}{
		"should succeed": {},
	}
	for name := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			logger, _ := zap.NewDevelopment()
			streamer, err := coinbase.NewClient(ctx, coinbase.WithLogger(logger))
			if err != nil {
				t.Fatalf("could not create new client")
			}

			service := NewService(ctx, streamer, WithLogger(logger), WithOutput(os.Stdout))
			service.AddTradingPairs("BTC-USD", "ETH-USD", "ETH-BTC")

			err = service.Run()
			fmt.Println(err)
			//if got := NewService(tt.args.ctx, tt.args.streamer, tt.args.opts...); !reflect.DeepEqual(got, tt.want) {
			//	t.Errorf("NewService() = %v, want %v", got, tt.want)
			//}
		})
	}
}
