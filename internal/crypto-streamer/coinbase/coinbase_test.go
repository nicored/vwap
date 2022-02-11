package coinbase

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"net/http"
	"testing"
)

var (
	errMsg  = Message{Type: TypeError, Message: "test error message"}
	subsMsg = Message{Type: TypeSubscriptions, Channels: Channels{
		NewChannel("matches", "BTC-USD", "ETH-BTC"),
	}}
	matches = []Message{
		{
			Type:      "match",
			Sequence:  1,
			ProductID: "ETH-BTC",
			Price:     "1.0",
			Size:      "0.1",
		},
		{
			Type:      "match",
			Sequence:  2,
			ProductID: "BTC-USD",
			Price:     "2.0",
			Size:      "0.2",
		},
		{
			Type:      "match",
			Sequence:  3,
			ProductID: "ETH-BTC",
			Price:     "3.0",
			Size:      "0.3",
		},
		{
			Type:      "match",
			Sequence:  4,
			ProductID: "BTC-USD",
			Price:     "4.0",
			Size:      "0.4",
		},
	}
)

func wsHandlerMock(t *testing.T, messages []Message) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		u := websocket.Upgrader{}
		c, err := u.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()

		for _, m := range messages {
			err = c.WriteJSON(m)
			if err != nil {
				break
			}
		}
	}
}

func TestNew(t *testing.T) {
	type args struct {
		ctx context.Context
		url string
	}
	tests := map[string]struct {
		args    args
		want    *WSClient
		wantErr bool
	}{
		"it should successfully subscribe": {
			args: args{
				ctx: context.Background(),
				url: _wsUrl,
			},
			want:    nil,
			wantErr: false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			//s := httptest.NewServer(http.HandlerFunc(wsHandlerMock()))
			//defer s.Close()

			//u := "ws" + strings.TrimPrefix(s.URL, "http")
			logger, err := zap.NewDevelopment()
			if err != nil {
				t.Fatal(err)
			}

			got, err := NewClient(tt.args.ctx, WithLogger(logger))
			if err != nil {
				t.Fatal(err)
			}
			defer got.conn.Close()

			go func() {
				//time.Sleep(20 * time.Second)
				err = got.Subscribe(Channels{
					NewChannel(ChannelMatches, "BTC-USD", "ETH-USD", "ETH-BTC", "BTC-EUR", "BTC-AUD", "ETH-EUR"),
				})
				if err != nil {
					t.Fatal(err)
				}

				//time.Sleep(5 * time.Second)
				//got.wsClient.conn.Close()
			}()

			feeds, done := got.Feeds()

		Selector:
			for {
				select {
				case <-done:
					break Selector
				case msg := <-feeds:
					data, err := json.MarshalIndent(msg, "", "  ")
					if err != nil {
						t.Fatal(err)
					}
					fmt.Println(string(data))
				}
			}
		})
	}
}
