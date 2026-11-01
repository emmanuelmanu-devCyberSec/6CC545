.PHONY: build test smoke vet race
build:
	go build ./...
test:
	go test ./... -count=1 -timeout 20s
smoke:
	go test ./... -run '^TestSmoke_' -count=1
vet:
	go vet ./...
race:
	go test -race ./... -count=1 -timeout 30s
