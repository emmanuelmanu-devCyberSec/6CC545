#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
command -v protoc >/dev/null 2>&1 || { echo "Install the module-approved protoc compiler first." >&2; exit 1; }
export GOBIN="$(pwd)/bin/tools"
mkdir -p "$GOBIN"
export PATH="$GOBIN:$PATH"
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
protoc --version
protoc-gen-go --version
protoc-gen-go-grpc --version
go generate ./api/files/v1
go mod tidy
go mod verify
echo "Bindings generated. Complete student TODOs, then run scripts/check.sh."
