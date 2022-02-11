package coinbase

import (
	"context"
	"fmt"
	ws "github.com/gorilla/websocket"
	"go.uber.org/zap"
	"strings"
	"time"
)

const (
	_reqLimitsPerSec = 100
	_wsUrl           = "wss://ws-feed.exchange.coinbase.com"
)

const (
	ChannelMatches = "matches"

	Subscribe   = "subscribe"
	Unsubscribe = "unsubscribe"

	TypeMatch         = "match"
	TypeError         = "error"
	TypeSubscriptions = "subscriptions"
	TypeLastMatch     = "last_match" // TypeLastMatch returned when we have missed trades after a disconnection
)

// Message represents the message object sent and expected by the websocket server
// The message Type dictates what properties are used in the request and response message. See API
// documentation for more information https://docs.cloud.coinbase.com/exchange/docs/websocket-overview
type Message struct {
	Type         string    `json:"type"`
	TradeID      int       `json:"trade_id,omitempty"`
	Sequence     int64     `json:"sequence,omitempty"`
	MakerOrderID string    `json:"maker_order_id"`
	TakerOrderID string    `json:"taker_order_id"`
	Time         time.Time `json:"time,omitempty"`
	ProductID    string    `json:"product_id,omitempty"`
	Size         string    `json:"size,omitempty"`
	Price        string    `json:"price,omitempty"`
	Message      string    `json:"message,omitempty"`
	Side         string    `json:"side,omitempty"`
	Channels     Channels  `json:"channels,omitempty"`
}

// WSClient is the Websocket client used by Coinbase to subscribe to channels
type WSClient struct {
	ctx    context.Context
	conn   *ws.Conn
	url    string
	logger *zap.Logger
}

// NewClient creates a new websocket client with an established connection
// to the websocket server
func NewClient(ctx context.Context, opts ...Option) (*WSClient, error) {
	options := options{
		logger: zap.NewNop(),
		wsUrl:  _wsUrl,
	}

	for _, o := range opts {
		o.apply(&options)
	}

	client := &WSClient{
		ctx:    ctx,
		url:    options.wsUrl,
		logger: options.logger,
	}

	if err := client.dial(); err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	return client, nil
}

// Subscribe subscribes to the provided channels and product ids
func (w *WSClient) Subscribe(channels Channels) error {
	reqMsg := Message{
		Type:     Subscribe,
		Channels: channels,
	}
	if err := w.conn.WriteJSON(reqMsg); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

// Unsubscribe unsubscribes from the provided channels and products ids
func (w *WSClient) Unsubscribe(channels Channels) error {
	reqMsg := Message{
		Type:     Unsubscribe,
		Channels: channels,
	}
	if err := w.conn.WriteJSON(reqMsg); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

// Close closes the connection to the server
func (w *WSClient) Close() error {
	if err := w.conn.Close(); err != nil {
		return fmt.Errorf("close connection: %w", err)
	}
	return nil
}

// Feeds sends new messages to the receiver channel
func (w *WSClient) Feeds() (feeds chan Message, done chan bool) {
	done = make(chan bool)
	feeds = make(chan Message)

	go func() {
		defer func() {
			done <- true
		}()

		// The server is rate limited to 100 requests / second per IP address
		// This limit could be reached when subscribing to several products with high amounts of trades.
		// We want to log whenever we exceed this limit and may decide to do something about it in the future if this
		// becomes an issue
		tick := time.Tick(1 * time.Second)
		nReqsPerSec := 0

		for {
			select {
			case <-tick:
				w.logger.Sugar().Debugf("websocket RPS: %d", nReqsPerSec)

				if nReqsPerSec >= _reqLimitsPerSec {
					w.logger.Sugar().Warnf("websocket rate limit reached: %d / 1 sec", nReqsPerSec)
				}

				nReqsPerSec = 0

			case <-w.ctx.Done():
				if err := w.conn.Close(); err != nil {
					w.logger.Sugar().Errorf("failed to close websocket connection: %v", err)
				}
				return

			default:
				subMsg := Message{}
				if err := w.conn.ReadJSON(&subMsg); err != nil {
					w.logger.Sugar().Errorf("failed to read message from server: %v", err)
					return
				}

				if subMsg.Type == TypeSubscriptions {
					w.logger.Info("subscription updated", zap.Any("channels", subMsg.Channels))
				}

				feeds <- subMsg
				nReqsPerSec++
			}
		}
	}()

	return
}

// dial establishes the connection to the websocket server
func (w *WSClient) dial() error {
	var err error

	w.conn, _, err = ws.DefaultDialer.DialContext(w.ctx, w.url, nil)
	if err != nil {
		return fmt.Errorf("dial ws server %s: %w", w.url, err)
	}

	return nil
}

// Channel represents a single element in the channels property in a Message
type Channel struct {
	Name       string   `json:"name"`
	ProductIDs []string `json:"product_ids,omitempty"`
}

// NewChannel creates a new channel
func NewChannel(name string, productIDS ...string) Channel {
	return Channel{
		Name:       name,
		ProductIDs: productIDS,
	}
}

// Channels represents the list of channels in a Message
type Channels []Channel

// String returns a human-readable format of the channels and their products ids
func (ch Channels) String() string {
	out := ""

	for i, channel := range ch {
		if i > 0 {
			out += " / "
		}
		out += channel.Name + "[" + strings.Join(channel.ProductIDs, ",") + "]"
	}

	return out
}
