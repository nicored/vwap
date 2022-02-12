package service

import (
	"go.uber.org/zap"
	"io"
	"os"
)

type options struct {
	logger     *zap.Logger
	maxDataPts int
	output     io.Writer
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
	if log == nil {
		log = zap.NewNop()
	}
	return loggerOption{Log: log}
}

type maxDataPtsOption struct {
	DataPoints int
}

func (m maxDataPtsOption) apply(opts *options) {
	opts.maxDataPts = m.DataPoints
}

func WithMaxDataPts(maxDataPts int) Option {
	if maxDataPts == 0 {
		maxDataPts = _defaultMaxPts
	}
	return maxDataPtsOption{DataPoints: maxDataPts}
}

type outputOption struct {
	output io.Writer
}

func (o outputOption) apply(opts *options) {
	opts.output = o.output
}

func WithOutput(output io.Writer) Option {
	if output == nil {
		output = os.Stdout
	}
	return outputOption{output: output}
}
