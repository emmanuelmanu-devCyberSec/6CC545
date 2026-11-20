package rpcfiles

import (
	"context"
	"fmt"

	pb "example.edu/6cc545/cw1/api/files/v1"
	"example.edu/6cc545/cw1/internal/config"
	"example.edu/6cc545/cw1/internal/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type Catalog interface {
	Snapshot() []model.FileMetadata
}

type Service struct {
	pb.UnimplementedFileServiceServer
	NodeID              string
	Catalog             Catalog
	RequirePeerIdentity bool
}

// verifyPeerIdentity checks that the request's claimed node ID matches
// the identity in the verified client certificate.
func verifyPeerIdentity(ctx context.Context, requesterID string) error {
	remotePeer, ok := peer.FromContext(ctx)
	if !ok || remotePeer.AuthInfo == nil {
		return status.Error(codes.Unauthenticated, "verified peer certificate required")
	}

	tlsInfo, ok := remotePeer.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return status.Error(codes.Unauthenticated, "TLS peer authentication required")
	}

	if len(tlsInfo.State.VerifiedChains) == 0 ||
		len(tlsInfo.State.PeerCertificates) == 0 {
		return status.Error(codes.Unauthenticated, "verified client certificate required")
	}

	expectedIdentity := "spiffe://cw2.local/" + requesterID

	for _, uri := range tlsInfo.State.PeerCertificates[0].URIs {
		if uri.String() == expectedIdentity {
			return nil
		}
	}

	return status.Error(codes.PermissionDenied, "requester_id does not match certificate identity")
}

func (s *Service) ListFiles(
	ctx context.Context,
	r *pb.ListFilesRequest,
) (*pb.ListFilesResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	if r == nil || !config.ValidID(r.GetRequesterId()) {
		return nil, status.Error(codes.InvalidArgument, "invalid requester_id")
	}

	// Production instances require the claimed requester ID to match
	// the identity authenticated by the mTLS connection.
	if s.RequirePeerIdentity {
		if err := verifyPeerIdentity(ctx, r.GetRequesterId()); err != nil {
			return nil, err
		}
	}

	if s.Catalog == nil {
		return nil, status.Error(codes.Internal, "file catalogue unavailable")
	}

	metadata := s.Catalog.Snapshot()
	files := make([]*pb.FileMetadata, 0, len(metadata))

	for _, f := range metadata {
		files = append(files, &pb.FileMetadata{
			Name:         f.Name,
			Size:         f.Size,
			Sha256:       f.SHA256,
			ModifiedUnix: f.ModifiedUnix,
		})
	}

	return &pb.ListFilesResponse{
		Files:  files,
		NodeId: s.NodeID,
	}, nil
}

var _ = fmt.Sprintf
