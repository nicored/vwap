package main

import (
	"context"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"vwap-service/internal/crypto-streamer/coinbase"
	"vwap-service/internal/service"
)

const (
	_envAppEnv         = "ENV"
	_envOutputPath     = "OUTPUT_PATH"
	_envTradingPairs   = "TRADING_PAIRS"
	_defaultOutput     = "/tmp/vwaps.txt"
	_appEnvDevelopment = "dev"
)

type Config struct {
	dev          bool
	outputPath   string
	tradingPairs []string
}

func main() {
	config := initConfig()
	ctx, cancelFunc := context.WithCancel(context.Background())

	logger := initLogger(config.dev)

	// create new file
	output, err := os.OpenFile(config.outputPath, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		panic(err)
	}
	defer output.Close()

	// prepare new exchange client
	streamer, err := coinbase.NewClient(ctx, coinbase.WithLogger(logger))
	if err != nil {
		panic(err)
	}
	defer streamer.Close()

	// prepare engine
	engine := service.NewService(ctx, streamer, service.WithLogger(logger), service.WithOutput(output))
	engine.AddTradingPairs(config.tradingPairs...)

	// run engine
	go func() {
		if err := engine.Run(); err != nil {
			panic(err)
		}
	}()

	termChan := make(chan os.Signal)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	<-termChan
	cancelFunc()
}

func initConfig() Config {
	return Config{
		dev:          isDev(),
		outputPath:   getOutputPath(),
		tradingPairs: getTradingPairs(),
	}
}

func initLogger(isDev bool) *zap.Logger {
	var err error
	var logger *zap.Logger

	if isDev {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}

	if err != nil {
		panic(err)
	}

	return logger
}

func isDev() bool {
	env, ok := os.LookupEnv(_envAppEnv)
	return !ok || env == _appEnvDevelopment
}

func getOutputPath() string {
	path, ok := os.LookupEnv(_envOutputPath)
	if !ok {
		return _defaultOutput
	}

	return path
}

func getTradingPairs() []string {
	tradingPairs, ok := os.LookupEnv(_envTradingPairs)
	if !ok {
		return []string{"BTC-USD", "ETH-USD", "ETH-BTC"}
	}
	return strings.Split(tradingPairs, ",")
}
