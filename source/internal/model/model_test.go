package model

import (
	"strings"
	"testing"
)

func TestWeek03_MetadataValidation(t *testing.T) {
	good := FileMetadata{Name: "notes.txt", Size: 1, SHA256: strings.Repeat("0", 64), ModifiedUnix: 1}
	if e := good.Validate(); e != nil {
		t.Fatal(e)
	}
	cases := []FileMetadata{good, good, good, good}
	cases[0].Name = "../escape"
	cases[1].Size = -1
	cases[2].SHA256 = "not-a-hash"
	cases[3].ModifiedUnix = -1
	for _, f := range cases {
		if f.Validate() == nil {
			t.Fatal("invalid metadata accepted")
		}
	}
}

func TestPeerValidation(t *testing.T) {
	// A peer advertised through a Docker service name should be accepted.
	good := Peer{
		ID:          "node-b",
		HTTPAddress: "node-b:8080",
		GRPCAddress: "node-b:9090",
		Status:      "online",
	}

	if err := good.Validate(); err != nil {
		t.Fatalf("valid peer was rejected: %v", err)
	}

	// These addresses should not be accepted as advertised peer addresses.
	cases := []Peer{
		{
			ID:          "node-b",
			HTTPAddress: "node-b:8080",
			GRPCAddress: "node-b:0",
			Status:      "online",
		},
		{
			ID:          "node-b",
			HTTPAddress: "http://node-b:8080",
			GRPCAddress: "node-b:9090",
			Status:      "online",
		},
		{
			ID:          "node-b",
			HTTPAddress: "node-b:8080",
			GRPCAddress: "bad host:9090",
			Status:      "online",
		},
	}

	for i, peer := range cases {
		if err := peer.Validate(); err == nil {
			t.Errorf("invalid peer case %d was accepted", i)
		}
	}
}
