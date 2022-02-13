FROM golang:1.16 AS base

COPY . /build
WORKDIR /build

RUN go test -v $(go list ./... | grep -v /vendor/) && \
    go build -o vwap main.go

ENTRYPOINT ["/bin/bash", "-c", "go test -v $(go list ./... | grep -v /vendor/)"]

FROM debian AS final

ENV ENV="production"
ENV OUTPUT_PATH="/tmp/vwap.txt"
ENV TRADING_PAIRS="BTC-USD,ETH-USD,ETH-BTC"

RUN apt-get update && apt-get -y install ca-certificates

COPY --from=base /build/vwap /app/vwap
WORKDIR /app

ENTRYPOINT "./vwap"