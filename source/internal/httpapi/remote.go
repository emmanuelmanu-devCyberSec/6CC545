package httpapi

import (
	"errors"
	"net/http"
	"time"

	pb "example.edu/6cc545/cw1/api/files/v1"
	"example.edu/6cc545/cw1/internal/config"
	"example.edu/6cc545/cw1/internal/model"
	"example.edu/6cc545/cw1/internal/rpcfiles"
	"example.edu/6cc545/cw1/internal/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RemoteFiles preserves the original teaching client for existing tests.
func RemoteFiles(reg *state.Registry, nodeID string, timeout time.Duration) http.HandlerFunc {
	return remoteFiles(reg, nodeID, timeout, func(r *http.Request, address, requester string) (*pb.ListFilesResponse, error) {
		return rpcfiles.Fetch(r.Context(), address, requester, timeout)
	})
}

// RemoteFilesSecure uses mutual TLS for production remote file requests.
func RemoteFilesSecure(
	reg *state.Registry,
	nodeID string,
	timeout time.Duration,
	certFile string,
	keyFile string,
	caFile string,
) http.HandlerFunc {
	return remoteFiles(reg, nodeID, timeout, func(r *http.Request, address, requester string) (*pb.ListFilesResponse, error) {
		// Use the requested peer ID as the expected TLS server identity.
		return rpcfiles.FetchSecure(
			r.Context(),
			address,
			requester,
			timeout,
			certFile,
			keyFile,
			caFile,
			requester,
		)
	})
}

// remoteFiles contains the shared HTTP handling for secure and legacy requests.
func remoteFiles(
	reg *state.Registry,
	nodeID string,
	timeout time.Duration,
	fetch func(*http.Request, string, string) (*pb.ListFilesResponse, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow GET requests only.
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			ErrorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}

		// Obtain the peer_id query value and reject a missing or malformed ID.
		peerID := r.URL.Query().Get("peer_id")
		if !config.ValidID(peerID) {
			ErrorJSON(w, http.StatusBadRequest, "invalid_peer_id")
			return
		}

		// Look up the peer in the registry; return 404 if it is missing.
		peer, err := reg.Get(peerID)
		if err != nil {
			if errors.Is(err, state.ErrMissing) {
				ErrorJSON(w, http.StatusNotFound, "peer_not_found")
				return
			}
			ErrorJSON(w, http.StatusInternalServerError, "internal_error")
			return
		}

		// Fetch the remote catalogue using the selected secure or legacy client.
		response, err := fetch(r, peer.GRPCAddress, nodeID)
		if err != nil {
			// Map remote gRPC errors to the required HTTP status codes.
			switch status.Code(err) {
			case codes.InvalidArgument:
				ErrorJSON(w, http.StatusBadRequest, "invalid_request")
			case codes.Unavailable:
				ErrorJSON(w, http.StatusServiceUnavailable, "remote_unavailable")
			case codes.DeadlineExceeded:
				ErrorJSON(w, http.StatusGatewayTimeout, "remote_timeout")
			default:
				// Map unexpected RPC errors consistently to 502 Bad Gateway.
				ErrorJSON(w, http.StatusBadGateway, "remote_error")
			}
			return
		}

		// Reject a missing response rather than treating it as an empty catalogue.
		if response == nil {
			ErrorJSON(w, http.StatusBadGateway, "remote_error")
			return
		}

		// Convert the protobuf file metadata into the shared REST model.
		files := make([]model.FileMetadata, 0, len(response.GetFiles()))
		for _, file := range response.GetFiles() {
			if file == nil {
				ErrorJSON(w, http.StatusBadGateway, "remote_error")
				return
			}
			files = append(files, model.FileMetadata{
				Name:         file.GetName(),
				Size:         file.GetSize(),
				SHA256:       file.GetSha256(),
				ModifiedUnix: file.GetModifiedUnix(),
			})
		}

		// Return JSON containing the remote node ID and its file metadata.
		WriteJSON(w, http.StatusOK, struct {
			NodeID string               `json:"node_id"`
			Files  []model.FileMetadata `json:"files"`
		}{
			NodeID: response.GetNodeId(),
			Files:  files,
		})
	}
}
