package security

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

var ErrNotImplemented = fmt.Errorf("TODO W09: security control not implemented")
var ErrUnauthenticated = fmt.Errorf("unauthenticated")
var ErrForbidden = fmt.Errorf("forbidden")

// loadCAPool reads the CA certificate file and prepares it for TLS verification.
func loadCAPool(caFile string) (*x509.CertPool, error) {
	if strings.TrimSpace(caFile) == "" {
		return nil, fmt.Errorf("CA certificate file path is required")
	}

	data, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("could not read CA certificate file: %w", err)
	}

	// This pool will contain the CA certificates we trust.
	caPool := x509.NewCertPool()
	certCount := 0

	for len(data) > 0 {
		data = bytesTrimSpace(data)
		if len(data) == 0 {
			break
		}

		// Read the next PEM block from the file.
		block, remaining := pem.Decode(data)
		if block == nil {
			return nil, fmt.Errorf("CA file contains invalid certificate data")
		}

		if block.Type != "CERTIFICATE" {
			return nil, fmt.Errorf("CA file contains a PEM block that is not a certificate")
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("could not parse CA certificate: %w", err)
		}

		// Only CA certificates should be added to the trust pool.
		if !cert.IsCA {
			return nil, fmt.Errorf("certificate in CA file is not a CA certificate")
		}

		caPool.AddCert(cert)
		certCount++
		data = remaining
	}

	if certCount == 0 {
		return nil, fmt.Errorf("CA file does not contain any certificates")
	}

	return caPool, nil
}

// Remove whitespace around PEM blocks before reading the next certificate.
func bytesTrimSpace(data []byte) []byte {
	return []byte(strings.TrimSpace(string(data)))
}

func ServerTLS(certFile, keyFile, caFile string, requireClient bool) (*tls.Config, error) {
	// A server must have its own certificate and private key.
	if strings.TrimSpace(certFile) == "" || strings.TrimSpace(keyFile) == "" {
		return nil, fmt.Errorf("server certificate and private key paths are required")
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("could not load server certificate and private key: %w", err)
	}

	// Load the CA that the server will use when checking certificates.
	caPool, err := loadCAPool(caFile)
	if err != nil {
		return nil, fmt.Errorf("could not configure server CA trust: %w", err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS12,
	}

	// Peer connections must present a valid certificate from our trusted CA.
	// Other HTTPS services can use server authentication without a client certificate.
	if requireClient {
		config.ClientAuth = tls.RequireAndVerifyClientCert
	} else {
		config.ClientAuth = tls.NoClientCert
	}

	return config, nil
}

func ClientTLS(certFile, keyFile, caFile, serverName string) (*tls.Config, error) {
	// The client needs its own certificate and the expected server name.
	if strings.TrimSpace(certFile) == "" || strings.TrimSpace(keyFile) == "" {
		return nil, fmt.Errorf("client certificate and private key paths are required")
	}

	if strings.TrimSpace(serverName) == "" {
		return nil, fmt.Errorf("expected server name is required")
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("could not load client certificate and private key: %w", err)
	}

	// Only certificates issued by this CA will be trusted.
	caPool, err := loadCAPool(caFile)
	if err != nil {
		return nil, fmt.Errorf("could not configure client CA trust: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		ServerName:   strings.TrimSpace(serverName),
		MinVersion:   tls.VersionTLS12,
	}, nil
}
