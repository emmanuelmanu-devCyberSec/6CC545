#!/usr/bin/env sh
set -eu
# Run from source/. First argument: smoke, acceptance, or race.
mode=${1:-smoke}
go version
go env GOOS GOARCH CGO_ENABLED CC GOMOD
case "$mode" in
 smoke) go test ./internal/events ./internal/storage -count=1 ;;
 acceptance) go test -tags acceptance ./internal/events ./internal/storage -count=1 -timeout 30s ;;
 race) go test -race -tags acceptance ./internal/events ./internal/storage -count=1 -timeout 60s ;;
 *) echo "usage: sh scripts/check-cw2.sh smoke|acceptance|race" >&2; exit 2 ;;
esac