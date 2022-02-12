package service

import (
	"vwap-service/internal/crypto-streamer/coinbase"
	"vwap-service/internal/vwap"
)

type ExchangeMsg struct {
	Type      string `json:"type"`
	ProductID string `json:"product_id"`
	Size      string `json:"size"`
	Price     string `json:"price"`
	Side      string `json:"side"`
}

type VWaper interface {
	Value() float64
	NPoints() int
	Push(price float64, volume float64) error
}

var _ VWaper = (*vwap.VWAP)(nil)

type Streamer interface {
	Subscribe(channel string, productIDs ...string) error
	Unsubscribe(channel string, productID ...string) error
	Feeds() (feeds chan []byte, feedsErr chan error)
	Close() error
}

var _ Streamer = (*coinbase.WSClient)(nil)

type Servicer interface {
	Run() error
	AddTradingPairs(pairs ...string)
	Stop()
}

var _ Servicer = (*Service)(nil)
