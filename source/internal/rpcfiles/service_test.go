package rpcfiles

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	pb "example.edu/6cc545/cw1/api/files/v1"
	"example.edu/6cc545/cw1/internal/model"
	"example.edu/6cc545/cw1/internal/state"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestWeek05_RemoteList(t *testing.T) {
	cat := state.NewCatalog()
	if e := cat.Replace([]model.FileMetadata{{Name: "notes.txt", Size: 1, SHA256: strings.Repeat("a", 64), ModifiedUnix: 1}}); e != nil {
		t.Fatal(e)
	}
	ln, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	s := grpc.NewServer()
	pb.RegisterFileServiceServer(s, &Service{NodeID: "node-b", Catalog: cat})
	done := make(chan struct{})
	go func() { defer close(done); _ = s.Serve(ln) }()
	defer func() { s.Stop(); _ = ln.Close(); <-done }()
	got, e := Fetch(context.Background(), ln.Addr().String(), "node-a", time.Second)
	if e != nil {
		t.Fatal(e)
	}
	if got.GetNodeId() != "node-b" || len(got.GetFiles()) != 1 || got.Files[0].GetName() != "notes.txt" {
		t.Fatalf("wrong remote result: %v", got)
	}
	_, e = Fetch(context.Background(), ln.Addr().String(), "", time.Second)
	if status.Code(e) != codes.InvalidArgument {
		t.Fatalf("empty requester: want InvalidArgument got %v", e)
	}
}
func TestWeek05_Unavailable(t *testing.T) {
	ln, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	start := time.Now()
	_, e = Fetch(context.Background(), addr, "node-a", 250*time.Millisecond)
	if e == nil {
		t.Fatal("unavailable peer returned success")
	}
	if status.Code(e) != codes.Unavailable && status.Code(e) != codes.DeadlineExceeded {
		t.Fatalf("unexpected status: %v", e)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("unbounded RPC")
	}
}
func TestWeek05_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e := (&Service{NodeID: "node-b", Catalog: state.NewCatalog()}).ListFiles(ctx, &pb.ListFilesRequest{RequesterId: "node-a"})
	if status.Code(e) != codes.Canceled {
		t.Fatalf("want Canceled got %v", e)
	}
}

func TestWeek05_ServiceInvalidRequesterAndEmptyCatalogue(t *testing.T) {
	// Create a service with an empty catalogue and a known node ID.
	svc := &Service{
		NodeID:  "node-b",
		Catalog: state.NewCatalog(),
	}

	// An empty requester ID should be rejected as an invalid argument.
	_, err := svc.ListFiles(context.Background(), &pb.ListFilesRequest{
		RequesterId: "",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("empty requester: want InvalidArgument, got %v", err)
	}

	// A valid requester should receive a successful response for an empty catalogue.
	got, err := svc.ListFiles(context.Background(), &pb.ListFilesRequest{
		RequesterId: "node-a",
	})
	if err != nil {
		t.Fatalf("empty catalogue: unexpected error: %v", err)
	}

	// The response should identify the node providing the file list.
	if got.GetNodeId() != "node-b" {
		t.Fatalf("node ID: want node-b, got %q", got.GetNodeId())
	}

	// An empty catalogue should return zero files, not an error.
	if len(got.GetFiles()) != 0 {
		t.Fatalf("empty catalogue: want 0 files, got %d", len(got.GetFiles()))
	}
}

func TestWeek05_ServiceMapsAllMetadataFields(t *testing.T) {
	// Create a catalogue containing one file with known metadata.
	cat := state.NewCatalog()
	err := cat.Replace([]model.FileMetadata{
		{
			Name:         "notes.txt",
			Size:         123,
			SHA256:       strings.Repeat("a", 64),
			ModifiedUnix: 1700000000,
		},
	})
	if err != nil {
		t.Fatalf("create catalogue metadata: %v", err)
	}

	// Call the service directly so the test does not depend on Fetch.
	svc := &Service{
		NodeID:  "node-b",
		Catalog: cat,
	}
	got, err := svc.ListFiles(context.Background(), &pb.ListFilesRequest{
		RequesterId: "node-a",
	})
	if err != nil {
		t.Fatalf("ListFiles returned an unexpected error: %v", err)
	}

	// Check that the response includes the service's node ID and one file.
	if got.GetNodeId() != "node-b" {
		t.Fatalf("node ID: want node-b, got %q", got.GetNodeId())
	}
	if len(got.GetFiles()) != 1 {
		t.Fatalf("file count: want 1, got %d", len(got.GetFiles()))
	}

	// Check that every metadata field was copied correctly.
	file := got.GetFiles()[0]
	if file.GetName() != "notes.txt" {
		t.Errorf("name: want notes.txt, got %q", file.GetName())
	}
	if file.GetSize() != 123 {
		t.Errorf("size: want 123, got %d", file.GetSize())
	}
	if file.GetSha256() != strings.Repeat("a", 64) {
		t.Errorf("SHA-256 value was not copied correctly")
	}
	if file.GetModifiedUnix() != 1700000000 {
		t.Errorf("modified timestamp: want 1700000000, got %d", file.GetModifiedUnix())
	}
}
