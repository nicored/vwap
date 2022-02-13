# VWAP calculation engine

Implementation of a real-time VWAP (volume-weighted average price) calculation engine using the coinbase websocket feed
to stream in trade executions and update the VWAP for each trading pair as updates become available.

## Test, build and run

```shell
# Build and run tests
docker build --target base -t nico/vwap-test:latest .
docker run --rm --name nico-vwap-tests -it nico/vwap-test

# Build and run service
docker build --target final -t nico/vwap:latest .
docker run --rm --name nico-vwap -it nico/vwap

# Run service in development mode
docker run --rm --name nico-vwap -e ENV=dev -it nico/vwap

# Tail results
docker exec -it nico-vwap tail -f /tmp/vwap.txt
```

## Design

**Exchange client**

The exchange client is not responsible for the business logic, and its only purpose is to fetch and
retrieve data from the exchange server from the subscribed channels and trading pairs requested by the main service,
and which can then interpret and process them to respect the business requirements.

**VWAP calculator**

The VWAP calculator was designed to handle one trading-pair by pushing in new entries and computing the
VWAP and storing it so it can easily be retrieved. The management of multiple trading-pairs is part of 
the core business logic and therefore implement in the main service.

**Service**

The main service's responsibility is to call the exchange client to fetch new matches, and compute the VWAPS
for all distinctive trading-pair matches fed by the exchange client. The service also writes updated VWAPs
to the provided writer.

**Output & Logs**

The VWAP outputs are written into a file, by default the file is located at `/tmp/vwap.txt`.
Error logs are written to stdout. DEBUG messages are logged only in `dev` mode.

## Config

Config values are extracted from environment variables, and which can be set on `docker run`

```shell
# app environment
ENV=dev

# output path for vwap updates
OUTPUT_PATH=/tmp/vwap.txt

# trading pairs to process
TRADING_PAIRS=BTC-USD,ETH-USD,ETH-BTC
```
