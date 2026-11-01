#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
go version
go build ./...
go test ./... -count=1 -timeout 20s
go vet ./...
echo "Checks passed. This is not a mark or proof of complete test coverage."
