package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"example.edu/6cc545/cw1/internal/config"
	"example.edu/6cc545/cw1/internal/security"
	"example.edu/6cc545/cw1/internal/storage"
)

// FetchPeerFile retrieves a file from a peer over HTTPS using mutual TLS.
// It verifies the server certificate, limits the response size, and checks
// the received content against the SHA-256 digest provided by the peer.
func FetchPeerFile(
	ctx context.Context,
	peerAddress string,
	fileName string,
	timeout time.Duration,
	certFile string,
	keyFile string,
	caFile string,
	serverName string,
) ([]byte, string, error) {
	if !config.ValidHTTPSPeerAddress(peerAddress) {
		return nil, "", fmt.Errorf("invalid HTTPS peer address")
	}

	if err := storage.ValidateName(fileName); err != nil {
		return nil, "", fmt.Errorf("invalid file name: %w", err)
	}

	if timeout <= 0 {
		return nil, "", fmt.Errorf("invalid transfer timeout")
	}

	// Configure TLS using the trusted CA and this node's client certificate.
	tlsConfig, err := security.ClientTLS(
		certFile,
		keyFile,
		caFile,
		serverName,
	)
	if err != nil {
		return nil, "", fmt.Errorf("configure peer TLS: %w", err)
	}

	peerURL, err := url.Parse(peerAddress)
	if err != nil {
		return nil, "", fmt.Errorf("parse peer address: %w", err)
	}

	peerURL.Path = path.Join(
		peerURL.Path,
		"/v1/peer/files",
		fileName,
	)

	peerURL.RawQuery = ""
	peerURL.Fragment = ""

	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodGet,
		peerURL.String(),
		nil,
	)
	if err != nil {
		return nil, "", fmt.Errorf("create peer file request: %w", err)
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	defer client.CloseIdleConnections()

	response, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("peer file request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		// Do not expose internal server details from the response body.
		return nil, "", fmt.Errorf(
			"peer file request returned HTTP %d",
			response.StatusCode,
		)
	}

	const maxPeerFileSize = 10 << 20 // 10 MiB

	if response.ContentLength > maxPeerFileSize {
		return nil, "", fmt.Errorf("peer file exceeds maximum size")
	}

	data, err := io.ReadAll(
		io.LimitReader(response.Body, maxPeerFileSize+1),
	)
	if err != nil {
		return nil, "", fmt.Errorf("read peer file: %w", err)
	}

	if int64(len(data)) > maxPeerFileSize {
		return nil, "", fmt.Errorf("peer file exceeds maximum size")
	}

	expectedDigest := strings.TrimSpace(
		response.Header.Get("X-Content-SHA256"),
	)

	if expectedDigest == "" {
		return nil, "", fmt.Errorf("peer response missing SHA-256 digest")
	}

	actualDigest := sha256.Sum256(data)
	actualHex := hex.EncodeToString(actualDigest[:])

	if !strings.EqualFold(actualHex, expectedDigest) {
		return nil, "", fmt.Errorf("peer file integrity verification failed")
	}

	return data, actualHex, nil
}
