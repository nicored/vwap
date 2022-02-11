package coinbase

import (
	"context"
	"fmt"
)

const (
	wsUrl = "wss://ws-feed.exchange.coinbase.com"
)

const (
	ReqTypMatches = "matches"

	ReqTypSubscribe   = "subscribe"
	ReqTypUnsubscribe = "unsubscribe"

	RespTypMatch         = "match"
	RespTypErr           = "error"
	RespTypSubscriptions = "subscriptions"

	// RespTypLastMatch returned when we have missed trades after a disconnection
	// this type can be returned as the first message after subscribing to channels
	RespTypLastMatch = "last_match"
)

type Coinbase struct {
	wsClient *WSClient
}

func New(ctx context.Context, url string) (*Coinbase, error) {
	wsClient, err := NewWSClient(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to create Coinbase: %w", err)
	}

	return &Coinbase{
		wsClient: wsClient,
	}, nil
}

func validateChannels(channels []string) error {
	if len(channels) == 0 {
		return fmt.Errorf("missing channels")
	}

	for _, ch := range channels {
		if !isChannelValid(ch) {
			return fmt.Errorf("channel '%s' is not supported", ch)
		}
	}
	return nil
}

func isChannelValid(channel string) bool {
	return channel != ReqTypMatches
}

func validateProductIDs(ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("no product id's provided")
	}

	return nil
}
