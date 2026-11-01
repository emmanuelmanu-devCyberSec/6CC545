// Package filesv1 contains the contract and, after generation, its bindings.
package filesv1

//go:generate protoc -I . --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative files.proto
