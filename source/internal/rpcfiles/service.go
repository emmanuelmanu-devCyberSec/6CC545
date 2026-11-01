package rpcfiles

import (
	"context"

	pb "example.edu/6cc545/cw1/api/files/v1"
	"example.edu/6cc545/cw1/internal/config"
	"example.edu/6cc545/cw1/internal/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Catalog interface{ Snapshot() []model.FileMetadata }
type Service struct {
	pb.UnimplementedFileServiceServer
	NodeID  string
	Catalog Catalog
}

func (s *Service) ListFiles(ctx context.Context, r *pb.ListFilesRequest) (*pb.ListFilesResponse, error) {
	// Return a gRPC error if the request context has been cancelled.
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	// Reject a missing request or an invalid requester ID.
	if r == nil || !config.ValidID(r.GetRequesterId()) {
		return nil, status.Error(codes.InvalidArgument, "invalid requester_id")
	}

	// Get the current catalogue snapshot without accessing the filesystem.
	metadata := s.Catalog.Snapshot()
	files := make([]*pb.FileMetadata, 0, len(metadata))

	// Convert each catalogue entry into its Protocol Buffers representation.
	for _, f := range metadata {
		files = append(files, &pb.FileMetadata{
			Name:         f.Name,
			Size:         f.Size,
			Sha256:       f.SHA256,
			ModifiedUnix: f.ModifiedUnix,
		})
	}

	// Return the file metadata and this service's node ID.
	return &pb.ListFilesResponse{
		Files:  files,
		NodeId: s.NodeID,
	}, nil
}

var _ = config.ValidID
