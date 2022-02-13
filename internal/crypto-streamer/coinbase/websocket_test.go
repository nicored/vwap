package coinbase

import (
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var (
	errMsg = Message{Type: TypeError, Message: "test error message"}

	channels = Channels{
		NewChannel("matches", "BTC-USD", "ETH-BTC"),
		NewChannel("heartbeat", "BTC-USD", "ETH-BTC"),
	}

	productIDs = []string{"BTC-USD", "ETH-BTC"}

	subscriptionsMsg = Message{Type: TypeSubscriptions, Channels: channels}

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

func echo(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		u := websocket.Upgrader{}
		c, err := u.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()

		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				break
			}
			err = c.WriteMessage(mt, message)
			if err != nil {
				break
			}
		}
	}
}

func wsTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	server := httptest.NewServer(echo(t))
	server.Client().Timeout = 200 * time.Millisecond
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	return server, url
}

func TestNewClient(t *testing.T) {
	type args struct {
		ctx  context.Context
		opts []Option
	}

	testLogger := zap.NewNop()

	tests := map[string]struct {
		args       args
		wantErr    assert.ErrorAssertionFunc
		wantLogger *zap.Logger
	}{
		"it should successfully create Client": {
			args: args{
				ctx:  context.Background(),
				opts: []Option{WithLogger(nil)},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return false
			},
		},
		"it should error when a wrong url is provided": {
			args: args{
				ctx:  context.Background(),
				opts: []Option{WithWSUrl("ws://this.will.throw.no.such.host")},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "dial: dial ws server ws://this.will.throw.no.such.host")
				return true
			},
		},
		"it should add a logger": {
			args: args{
				ctx:  context.Background(),
				opts: []Option{WithLogger(testLogger)},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return false
			},
			wantLogger: testLogger,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			server, wsUrl := wsTestServer(t)
			defer server.Close()

			// appends test server wsUrl to list of options
			tt.args.opts = append([]Option{WithWSUrl(wsUrl)}, tt.args.opts...)

			client, err := NewClient(tt.args.ctx, tt.args.opts...)
			if client != nil {
				defer client.Close()
			}

			tt.wantErr(t, err)
			if err != nil {
				return
			}

			assert.NotNil(t, client.conn, "connection should not be nil")
			assert.Equal(t, wsUrl, client.url, "they should match")

			if tt.wantLogger != nil {
				assert.Equal(t, tt.wantLogger, client.logger, "they should point to the same logger")
			}
		})
	}
}

func TestWSClient_Subscribe(t *testing.T) {
	type args struct {
		channels Channels
	}
	tests := map[string]struct {
		args      args
		wantErr   assert.ErrorAssertionFunc
		messages  []Message
		closeConn bool
	}{
		"it should subscribe successfully": {
			args: args{
				channels: channels,
			},
			messages: []Message{subscriptionsMsg},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return false
			},
			closeConn: false,
		},
		"it should error when unsubscribing": {
			args: args{
				channels: channels,
			},
			messages: []Message{subscriptionsMsg},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return true
			},
			closeConn: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			server, wsUrl := wsTestServer(t)
			defer server.Close()

			w, err := NewClient(context.Background(), WithWSUrl(wsUrl))
			if err != nil {
				t.Fatalf("client should not be nil")
			}
			defer w.Close()

			// closing the connection should throw an error
			if tt.closeConn {
				w.Close()
			}

			tt.wantErr(t, w.Subscribe(ChannelMatches, productIDs...), "Subscribe(matches)")

			if !tt.closeConn {
				serverIncomingMsg := Message{}
				err = w.conn.ReadJSON(&serverIncomingMsg)
				if err != nil {
					t.Fatalf("read server incoming: unexpected error: %v", err)
				}

				assert.Equal(t, serverIncomingMsg, Message{
					Type: Subscribe,
					Channels: Channels{
						Channel{Name: ChannelMatches, ProductIDs: productIDs},
					},
				})
			}
		})
	}
}

func TestWSClient_Unsubscribe(t *testing.T) {
	type args struct {
		channels Channels
	}
	tests := map[string]struct {
		args      args
		messages  []Message
		wantErr   assert.ErrorAssertionFunc
		closeConn bool
	}{
		"it should unsubscribe successfully": {
			args: args{
				channels: channels,
			},
			messages: []Message{subscriptionsMsg},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return false
			},
		},
		"it should error when unsubscribing": {
			args: args{
				channels: channels,
			},
			messages: []Message{subscriptionsMsg},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return true
			},
			closeConn: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			server, wsUrl := wsTestServer(t)
			defer server.Close()

			w, err := NewClient(context.Background(), WithWSUrl(wsUrl))
			if err != nil {
				t.Fatalf("client should not be nil")
			}
			defer w.Close()

			// closing the connection should throw an error
			if tt.closeConn {
				w.Close()
			}

			tt.wantErr(t, w.Unsubscribe(ChannelMatches, productIDs...), "Unsubscribe()")

			if !tt.closeConn {
				serverIncomingMsg := Message{}
				err = w.conn.ReadJSON(&serverIncomingMsg)
				if err != nil {
					t.Fatalf("read server incoming: unexpected error: %v", err)
				}

				assert.Equal(t, serverIncomingMsg, Message{
					Type: Unsubscribe,
					Channels: Channels{
						Channel{Name: "matches", ProductIDs: productIDs},
					},
				})
			}
		})
	}
}

func TestWSClient_Feeds_should_succeed(t *testing.T) {
	tests := map[string]struct {
		messages []Message
		logger   *zap.Logger
	}{
		"should successfully return messages": {
			messages: matches[:2],
		},
		"should successfully subscribe": {
			messages: []Message{subscriptionsMsg},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			server, wsUrl := wsTestServer(t)
			defer server.Close()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			observedZapCore, observedLogs := observer.New(zap.InfoLevel)
			observedLogger := zap.New(observedZapCore)

			w, err := NewClient(ctx, WithWSUrl(wsUrl), WithLogger(observedLogger))
			if err != nil {
				t.Fatalf("client should not be nil")
			}
			defer w.Close()

			feeds, errFeeds := w.Feeds()

			for _, m := range tt.messages {
				w.conn.WriteJSON(m)

				select {
				case <-time.Tick(1 * time.Second):
					t.Fatalf("timed out waiting for feed")
				case msg := <-feeds:
					subMsg := Message{}
					if err = json.Unmarshal(msg, &subMsg); err != nil {
						t.Fatalf("error unmarshalling response")
					}

					// assert that we log subscriptions
					allLogs := observedLogs.All()
					if subMsg.Type == TypeSubscriptions {
						assert.Equal(t, "subscription updated", allLogs[0].Message)
						assert.ElementsMatch(t, []zap.Field{
							{Key: "channels", Type: zapcore.StringerType, Interface: channels},
						}, allLogs[0].Context)
					} else {
						assert.Len(t, allLogs, 0)
					}

					assert.Equal(t, m, subMsg)
				case feedErr := <-errFeeds:
					t.Errorf("did not expect an error. got %v", feedErr)
				}
			}

		})
	}
}

func TestWSClient_Feeds_should_error_when_connection_closed(t *testing.T) {
	server, wsUrl := wsTestServer(t)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w, err := NewClient(ctx, WithWSUrl(wsUrl))
	if err != nil {
		t.Fatalf("client should not be nil")
	}

	_, errFeeds := w.Feeds()

	// close connection to generate an error
	w.Close()

	select {
	case <-time.Tick(1 * time.Second):
		t.Fatalf("timed out waiting for feed")
	case feedErr := <-errFeeds:
		assert.Contains(t, feedErr.Error(), "read message: read tcp")
	}
}
