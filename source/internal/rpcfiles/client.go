package rpcfiles

import (
	"context"
	"time"

	pb "example.edu/6cc545/cw1/api/files/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func Fetch(ctx context.Context, address, requester string, timeout time.Duration) (*pb.ListFilesResponse, error) {
	// Derive an operation context from the supplied caller context and timeout.
	// Use insecure transport credentials ONLY for this loopback teaching setup.
	// Do not call an empty/invalid address.
	if address == "" {
		return nil, status.Error(codes.InvalidArgument, "empty server address")
	}
	if timeout <= 0 {
		return nil, status.Error(codes.InvalidArgument, "timeout must be positive")
	}

	opCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create the client connection and configure bounded call message sizes.
	// NewClient does not prove the server is reachable: the RPC does the I/O.
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(4096),
			grpc.MaxCallRecvMsgSize(65536),
		),
	)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "create gRPC client: %v", err)
	}

	// Defer connection closure and invoke the generated client with the operation context.
	defer conn.Close()

	client := pb.NewFileServiceClient(conn)
	response, err := client.ListFiles(opCtx, &pb.ListFilesRequest{
		RequesterId: requester,
	})
	if err != nil {
		// Preserve cancellation and RPC errors instead of returning an empty list.
		return nil, err
	}

	return response, nil
}

var _ = grpc.NewClient
var _ = insecure.NewCredentials
