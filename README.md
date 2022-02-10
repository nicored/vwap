# VWAP calculation engine

## Overview

TODO: Write up

## Design

TODO: Write up

- Client interface & coinbase client
- VWaper interface & calculation engine
- MessageBroker interface & output to file
- Service to handler WS messages from coinbase client

## External libraries used
- testify...
- gorilla...
- ???

## Tasks breakdown

### VWAP Calculation Engine
- [ ] CalculateVWAP
- [ ] Tests

### Coinbase Websocket client (crypto-streamer)
- [ ] Interface (Subscribe, Unsubscribe)   
- [ ] Coinbase client
- [ ] Subscribe to channel & products
- [ ] Subscription tests
- [ ] Unsubscribe
- [ ] Unsubscription tests

### Message broker (mbroker)
- [ ] PublishCloser Interface
- [ ] SimpleBroker to writer
- [ ] Tests

### Service (service)
- [ ] New Service
- [ ] Start
- [ ] Stop
- [ ] Logger
- [ ] Graceful shutdown
- [ ] Tests

### Wrapper
- [ ] Main (hardcoded config)
- [ ] Parse & use config file
- [ ] Catch signal & graceful shutdown
- [ ] Tests

### Performance
- [ ] Benchmarking
- [ ] Memory leaks
- [ ] Documentation

### Build & scripts
- [ ] Multi-stage Dockerfile (code, tests, final)
- [ ] Scripts (test, build, run, logs, output)
- [ ] Documentation
