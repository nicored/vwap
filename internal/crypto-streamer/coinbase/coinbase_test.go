package coinbase

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"testing"
	"time"
)

var (
	errMsg  = Message{Type: RespTypErr, Message: "test error message"}
	matches = []Message{
		{
			Type:      "match",
			ProductID: "BTC-USD",
			Price:     "1",
			Size:      "0.1",
		},
		{
			Type:      "match",
			ProductID: "BTC-USD",
			Price:     "2",
			Size:      "0.2",
		},
		{
			Type:      "match",
			ProductID: "BTC-USD",
			Price:     "3",
			Size:      "0.3",
		},
		{
			Type:      "match",
			ProductID: "BTC-USD",
			Price:     "4",
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
		want    *Coinbase
		wantErr bool
	}{
		"success?": {
			args: args{
				ctx: context.Background(),
				url: wsUrl,
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
			got, err := New(tt.args.ctx, wsUrl)
			if err != nil {
				t.Fatal(err)
			}
			defer got.wsClient.conn.Close()

			go func() {
				time.Sleep(20 * time.Second)
				err = got.wsClient.Subscribe(Channels{
					NewChannel(ReqTypMatches, "BTC-USD", "ETH-USD", "ETH-BTC", "BTC-EUR", "BTC-AUD", "ETH-EUR"),
				})
				if err != nil {
					t.Fatal(err)
				}

				//time.Sleep(5 * time.Second)
				//got.wsClient.conn.Close()
			}()

			receiver := make(chan Message)
			done := got.wsClient.Feeds(receiver)

		Selector:
			for {
				select {
				case <-done:
					break Selector
				case msg := <-receiver:
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
