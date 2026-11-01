#!/bin/sh
set -eu
cd "$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
mkdir -p bin
go build -o bin/node ./cmd/node
exec ./bin/node -id node-a -http 127.0.0.1:8081 -seed testdata/node-a-files.json -tcp 127.0.0.1:7001 -udp 127.0.0.1:7101 -grpc 127.0.0.1:9091
