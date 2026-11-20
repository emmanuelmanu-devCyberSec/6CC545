//go:build acceptance

package rpcfiles

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/url"
	"testing"

	pb "example.edu/6cc545/cw1/api/files/v1"
	"example.edu/6cc545/cw1/internal/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// identityTestCatalog provides a catalogue for the service acceptance tests.
type identityTestCatalog struct{}

func (identityTestCatalog) Snapshot() []model.FileMetadata {
	return []model.FileMetadata{}
}

// identityTestContext creates a peer context representing a TLS connection.
// The verified-chain field represents a certificate already verified by TLS.
func identityTestContext(uri string, verified bool) context.Context {
	certificate := &x509.Certificate{}

	if uri != "" {
		parsedURI, err := url.Parse(uri)
		if err != nil {
			panic(err)
		}
		certificate.URIs = []*url.URL{parsedURI}
	}

	state := tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{certificate},
	}

	if verified {
		state.VerifiedChains = [][]*x509.Certificate{{certificate}}
	}

	tlsInfo := credentials.TLSInfo{State: state}

	return peer.NewContext(context.Background(), &peer.Peer{
		AuthInfo: tlsInfo,
	})
}

// TestListFilesAcceptsMatchingCertificateIdentity checks that a verified
// certificate with the matching node identity is accepted.
func TestListFilesAcceptsMatchingCertificateIdentity(t *testing.T) {
	service := &Service{
		NodeID:              "node-b",
		Catalog:             identityTestCatalog{},
		RequirePeerIdentity: true,
	}

	ctx := identityTestContext("spiffe://cw2.local/node-a", true)

	response, err := service.ListFiles(ctx, &pb.ListFilesRequest{
		RequesterId: "node-a",
	})

	if err != nil {
		t.Fatalf("expected matching identity to be accepted, got: %v", err)
	}

	if response == nil || response.GetNodeId() != "node-b" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

// TestListFilesRejectsMismatchedCertificateIdentity checks that a requester
// cannot claim a node ID different from the certificate identity.
func TestListFilesRejectsMismatchedCertificateIdentity(t *testing.T) {
	service := &Service{
		NodeID:              "node-b",
		Catalog:             identityTestCatalog{},
		RequirePeerIdentity: true,
	}

	ctx := identityTestContext("spiffe://cw2.local/node-b", true)

	_, err := service.ListFiles(ctx, &pb.ListFilesRequest{
		RequesterId: "node-a",
	})

	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got: %v", err)
	}
}

// TestListFilesRejectsMissingPeerCertificate checks that requests without
// TLS peer credentials cannot access the protected service.
func TestListFilesRejectsMissingPeerCertificate(t *testing.T) {
	service := &Service{
		NodeID:              "node-b",
		Catalog:             identityTestCatalog{},
		RequirePeerIdentity: true,
	}

	_, err := service.ListFiles(context.Background(), &pb.ListFilesRequest{
		RequesterId: "node-a",
	})

	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got: %v", err)
	}
}

// TestListFilesRejectsUnverifiedPeerCertificate checks that a certificate
// without a verified chain is not accepted.
func TestListFilesRejectsUnverifiedPeerCertificate(t *testing.T) {
	service := &Service{
		NodeID:              "node-b",
		Catalog:             identityTestCatalog{},
		RequirePeerIdentity: true,
	}

	ctx := identityTestContext("spiffe://cw2.local/node-a", false)

	_, err := service.ListFiles(ctx, &pb.ListFilesRequest{
		RequesterId: "node-a",
	})

	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got: %v", err)
	}
}
