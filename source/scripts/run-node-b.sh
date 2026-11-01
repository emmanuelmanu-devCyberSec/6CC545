#!/bin/sh
set -eu
cd "$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
mkdir -p bin
go build -o bin/node ./cmd/node
exec ./bin/node -id node-b -http 127.0.0.1:8082 -seed testdata/node-b-files.json -tcp 127.0.0.1:7002 -udp 127.0.0.1:7102 -grpc 127.0.0.1:9092
