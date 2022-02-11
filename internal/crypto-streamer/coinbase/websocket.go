package coinbase

import (
	"context"
	"fmt"
	ws "github.com/gorilla/websocket"
	"log"
	"strings"
	"time"
)

const (
	reqLimitsPerSec = 100
)

// Message represents the message object sent and expected by the websocket server
// The message Type dictates what properties are used in the request and response message. See API
// documentation for more information https://docs.cloud.coinbase.com/exchange/docs/websocket-overview
type Message struct {
	Type      string    `json:"type"`
	Message   string    `json:"message,omitempty"`
	TradeID   int       `json:"trade_id,omitempty"`
	Sequence  int64     `json:"sequence,omitempty"`
	ProductID string    `json:"product_id,omitempty"`
	Size      string    `json:"size,omitempty"`
	Price     string    `json:"price,omitempty"`
	Side      string    `json:"side,omitempty"`
	Time      time.Time `json:"time,omitempty"`
	Channels  Channels  `json:"channels,omitempty"`
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

// WSClient is the Websocket client used by Coinbase to subscribe to channels
type WSClient struct {
	ctx  context.Context
	conn *ws.Conn
	url  string
}

// NewWSClient creates a new websocket client with an established connection
// to the websocket server
func NewWSClient(ctx context.Context, url string) (*WSClient, error) {
	client := &WSClient{
		ctx: ctx,
		url: url,
	}

	if err := client.dial(); err != nil {
		return nil, fmt.Errorf("failed creating new client: %w", err)
	}

	return client, nil
}

// Subscribe subscribes to the provided channels and product ids
func (w *WSClient) Subscribe(channels Channels) error {
	reqMsg := Message{
		Type:     ReqTypSubscribe,
		Channels: channels,
	}
	if err := w.conn.WriteJSON(reqMsg); err != nil {
		return fmt.Errorf("could not subscribe to %s. %w", channels.String(), err)
	}

	return nil
}

// Unsubscribe unsubscribes from the provided channels and products ids
func (w *WSClient) Unsubscribe(channels Channels) error {
	reqMsg := Message{
		Type:     ReqTypUnsubscribe,
		Channels: channels,
	}
	if err := w.conn.WriteJSON(reqMsg); err != nil {
		return fmt.Errorf("failed to unsubscribe from %s. %w", channels.String(), err)
	}

	return nil
}

// Close closes the connection to the server
func (w *WSClient) Close() error {
	if err := w.conn.Close(); err != nil {
		return fmt.Errorf("failed closing the connection: %w", err)
	}
	return nil
}

// Feeds sends new messages to the receiver channel
func (w *WSClient) Feeds(receiver chan Message) chan bool {
	done := make(chan bool)
	go func() {
		// The server is rate limited to 100 requests / second per IP address
		// This limit could be reached when subscribing to several products with high amounts of trades.
		// We want to log whenever we exceed this limit and may decide to do something about it in the future if this
		// becomes an issue
		tick := time.Tick(1 * time.Second)
		nReqsPerSec := 0

		for {
			select {
			case <-tick:
				//log.Printf("%d reads / 1 second", nReqsPerSec)
				if nReqsPerSec >= reqLimitsPerSec {
					log.Printf("coinbase rate limit reached, %d reads / 1 second", nReqsPerSec)
				}
				nReqsPerSec = 0
			case <-w.ctx.Done():
				if err := w.conn.Close(); err != nil {
					log.Printf("could not close websocket connection. %v", err)
				}
				return
			default:
				subMsg := Message{}
				if err := w.conn.ReadJSON(&subMsg); err != nil {
					log.Printf("failed to read message from server: %v", err)
					done <- true
					return
				}

				if subMsg.Type == RespTypSubscriptions {
					log.Printf("subscribed to %s", subMsg.Channels.String())
				}

				receiver <- subMsg
				nReqsPerSec++
			}
		}
	}()
	return done
}

// dial establishes the connection to the websocket server
func (w *WSClient) dial() error {
	var err error

	w.conn, _, err = ws.DefaultDialer.DialContext(w.ctx, w.url, nil)
	if err != nil {
		return fmt.Errorf("failed to dial WS server %s", w.url)
	}

	return nil
}
