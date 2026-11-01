package model

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"example.edu/6cc545/cw1/internal/config"
)

type Peer struct {
	ID          string `json:"id"`
	HTTPAddress string `json:"http_address"`
	GRPCAddress string `json:"grpc_address"`
	Status      string `json:"status"`
}
type FileMetadata struct {
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	SHA256       string `json:"sha256"`
	ModifiedUnix int64  `json:"modified_unix"`
}

func (p Peer) Validate() error {
	if !config.ValidID(p.ID) {
		return fmt.Errorf("invalid peer ID %q", p.ID)
	}

	if !config.ValidAddress(p.HTTPAddress) {
		return errors.New("invalid peer HTTP address")
	}

	if !config.ValidAddress(p.GRPCAddress) {
		return errors.New("invalid peer gRPC address")
	}

	switch p.Status {
	case "unknown", "online", "offline":
		return nil
	default:
		return fmt.Errorf("invalid peer status %q", p.Status)
	}
}
func (f FileMetadata) Validate() error {
	if f.Name == "" || len(f.Name) > 128 {
		return errors.New("filename must contain 1 to 128 bytes")
	}

	if f.Name == "." || f.Name == ".." ||
		strings.ContainsAny(f.Name, "/\\\x00") {
		return errors.New("invalid filename")
	}

	if f.Size < 0 {
		return errors.New("file size cannot be negative")
	}

	if f.ModifiedUnix < 0 {
		return errors.New("modified timestamp cannot be negative")
	}

	if len(f.SHA256) != 64 {
		return errors.New("SHA-256 must contain 64 hexadecimal characters")
	}

	if _, err := hex.DecodeString(f.SHA256); err != nil {
		return errors.New("SHA-256 must contain 64 hexadecimal characters")
	}

	return nil
}

var _ = hex.DecodeString
var _ = strings.ContainsAny
var _ = config.ValidID
