package coinbase

import (
	"go.uber.org/zap"
)

type options struct {
	logger *zap.Logger
	wsUrl  string
}

type Option interface {
	apply(*options)
}

type loggerOption struct {
	Log *zap.Logger
}

func (l loggerOption) apply(opts *options) {
	opts.logger = l.Log
}

func WithLogger(log *zap.Logger) Option {
	return loggerOption{Log: log}
}

type wsUrlOption struct {
	Url string
}

func (u wsUrlOption) apply(opts *options) {
	opts.wsUrl = u.Url
}

func WithWSUrl(url string) Option {
	return wsUrlOption{Url: url}
}
