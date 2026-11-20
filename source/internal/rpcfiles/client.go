package rpcfiles

import (
	"context"
	"time"

	pb "example.edu/6cc545/cw1/api/files/v1"
	"example.edu/6cc545/cw1/internal/security"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// Fetch preserves the original loopback teaching client.
func Fetch(ctx context.Context, address, requester string, timeout time.Duration) (*pb.ListFilesResponse, error) {
	return fetch(ctx, address, requester, timeout, insecure.NewCredentials())
}

// FetchSecure connects using mutual TLS and verifies the expected server identity.
func FetchSecure(ctx context.Context, address, requester string, timeout time.Duration, certFile, keyFile, caFile, serverName string) (*pb.ListFilesResponse, error) {
	tlsConfig, err := security.ClientTLS(certFile, keyFile, caFile, serverName)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "configure gRPC TLS: %v", err)
	}

	return fetch(ctx, address, requester, timeout, credentials.NewTLS(tlsConfig))
}

func fetch(ctx context.Context, address, requester string, timeout time.Duration, transport credentials.TransportCredentials) (*pb.ListFilesResponse, error) {
	if address == "" {
		return nil, status.Error(codes.InvalidArgument, "empty server address")
	}
	if timeout <= 0 {
		return nil, status.Error(codes.InvalidArgument, "timeout must be positive")
	}

	opCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(transport),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(4096),
			grpc.MaxCallRecvMsgSize(65536),
		),
	)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "create gRPC client: %v", err)
	}
	defer conn.Close()

	client := pb.NewFileServiceClient(conn)
	response, err := client.ListFiles(opCtx, &pb.ListFilesRequest{
		RequesterId: requester,
	})
	if err != nil {
		return nil, err
	}

	return response, nil
}
