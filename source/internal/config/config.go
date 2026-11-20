package config

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Config holds the settings for one running node.
type Config struct {
	ID   string
	HTTP string
	TCP  string
	UDP  string
	GRPC string

	// Advertised addresses tell other nodes how to reach this node.
	HTTPAdvertise string
	GRPCAdvertise string

	// Secure peer HTTP listener and its advertised HTTPS endpoint.
	PeerHTTPAddr      string
	PeerHTTPAdvertise string

	// ContainerMode permits wildcard listener addresses inside Docker.
	// Local mode remains restricted to loopback addresses.
	ContainerMode bool

	DataDir   string
	PeerSeeds []string
	Seed      string
	Timeout   time.Duration

	JWTKeyFile  string
	JWTIssuer   string
	JWTAudience string

	TLSCertFile  string
	TLSKeyFile   string
	CAFile       string
	SecurityMode string
}

// Parse loads configuration using CLI > environment > local defaults.
func Parse(args []string, out io.Writer) (Config, error) {
	var c Config

	id := envOrDefault("NODE_ID", "node-a")
	httpAddr := envOrDefault("HTTP_ADDR", "127.0.0.1:8081")
	tcpAddr := os.Getenv("TCP_ADDR")
	udpAddr := os.Getenv("UDP_ADDR")
	grpcAddr := os.Getenv("GRPC_ADDR")

	httpAdvertise := os.Getenv("HTTP_ADVERTISE")
	grpcAdvertise := os.Getenv("GRPC_ADVERTISE")
	peerHTTPAddr := os.Getenv("PEER_HTTP_ADDR")
	peerHTTPAdvertise := os.Getenv("PEER_HTTP_ADVERTISE")

	dataDir := os.Getenv("DATA_DIR")
	peerSeeds := os.Getenv("PEER_SEEDS")
	metadataSeed := os.Getenv("SEED")

	jwtKeyFile := os.Getenv("JWT_KEY_FILE")
	jwtIssuer := os.Getenv("JWT_ISSUER")
	jwtAudience := os.Getenv("JWT_AUDIENCE")

	tlsCertFile := os.Getenv("TLS_CERT_FILE")
	tlsKeyFile := os.Getenv("TLS_KEY_FILE")
	caFile := os.Getenv("CA_FILE")
	securityMode := os.Getenv("SECURITY_MODE")

	containerMode, err := envBool("CONTAINER_MODE", false)
	if err != nil {
		return c, err
	}

	// Keep the timeout as text until after CLI parsing. This allows an
	// explicit CLI value to override an invalid environment value.
	timeoutText := envOrDefault("NETWORK_TIMEOUT", "2s")

	f := flag.NewFlagSet("node", flag.ContinueOnError)
	f.SetOutput(out)

	f.StringVar(&id, "id", id, "unique local node ID")
	f.StringVar(&httpAddr, "http", httpAddr, "HTTP listening address")
	f.StringVar(&tcpAddr, "tcp", tcpAddr, "optional TCP listening address")
	f.StringVar(&udpAddr, "udp", udpAddr, "optional UDP listening address")
	f.StringVar(&grpcAddr, "grpc", grpcAddr, "optional gRPC listening address")

	f.StringVar(&httpAdvertise, "http-advertise", httpAdvertise, "HTTP advertised address")
	f.StringVar(&grpcAdvertise, "grpc-advertise", grpcAdvertise, "gRPC advertised address")
	f.StringVar(&dataDir, "data-dir", dataDir, "local storage directory")
	f.StringVar(&peerSeeds, "peer-seeds", peerSeeds, "comma-separated peer addresses")
	f.StringVar(&metadataSeed, "seed", metadataSeed, "synthetic metadata JSON file")

	f.StringVar(&timeoutText, "timeout", timeoutText, "network operation timeout")
	f.BoolVar(&containerMode, "container", containerMode, "enable container listener validation")

	if err := f.Parse(args); err != nil {
		return c, err
	}

	if len(f.Args()) != 0 {
		return c, fmt.Errorf("unexpected arguments: %v", f.Args())
	}

	timeout, err := time.ParseDuration(timeoutText)
	if err != nil {
		return c, fmt.Errorf("invalid network timeout %q: %w", timeoutText, err)
	}

	c.ID = id
	c.HTTP = httpAddr
	c.TCP = tcpAddr
	c.UDP = udpAddr
	c.GRPC = grpcAddr
	c.HTTPAdvertise = httpAdvertise
	c.GRPCAdvertise = grpcAdvertise
	c.PeerHTTPAddr = peerHTTPAddr
	c.PeerHTTPAdvertise = peerHTTPAdvertise
	c.ContainerMode = containerMode
	c.Seed = metadataSeed
	c.Timeout = timeout

	c.JWTKeyFile = jwtKeyFile
	c.JWTIssuer = jwtIssuer
	c.JWTAudience = jwtAudience

	c.TLSCertFile = tlsCertFile
	c.TLSKeyFile = tlsKeyFile
	c.CAFile = caFile
	c.SecurityMode = securityMode

	if dataDir == "" {
		dataDir = filepath.Join("data", c.ID)
	}
	c.DataDir = dataDir
	c.PeerSeeds = splitPeerSeeds(peerSeeds)

	return c, c.Validate()
}

// envOrDefault returns an environment value or its fallback.
func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

// envBool reads an optional boolean environment variable.
func envBool(name string, fallback bool) (bool, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid %s value %q: %w", name, value, err)
	}
	return parsed, nil
}

// splitPeerSeeds parses a comma-separated list of peer endpoints.
func splitPeerSeeds(value string) []string {
	var seeds []string
	for _, seed := range strings.Split(value, ",") {
		seed = strings.TrimSpace(seed)
		if seed != "" {
			seeds = append(seeds, seed)
		}
	}
	return seeds
}

var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

func ValidID(s string) bool {
	return idPattern.MatchString(s)
}

// ValidAddress preserves the original loopback-only address policy.
func ValidAddress(s string) bool {
	h, p, err := net.SplitHostPort(s)
	if err != nil || h == "" {
		return false
	}

	n, err := strconv.Atoi(p)
	if err != nil || n < 1 || n > 65535 {
		return false
	}

	return h == "localhost" ||
		(net.ParseIP(h) != nil && net.ParseIP(h).IsLoopback())
}

// ValidListenAddress permits loopback and wildcard listener addresses.
// Config.Validate only uses this in explicit container mode.
func ValidListenAddress(s string) bool {
	h, p, err := net.SplitHostPort(s)
	if err != nil || h == "" {
		return false
	}

	n, err := strconv.Atoi(p)
	if err != nil || n < 1 || n > 65535 {
		return false
	}

	if h == "localhost" {
		return true
	}

	ip := net.ParseIP(h)
	return ip != nil && (ip.IsLoopback() || ip.IsUnspecified())
}

// ValidAdvertisedAddress validates a peer endpoint's host and port.
// Wildcard addresses are never valid advertised destinations.
func ValidAdvertisedAddress(s string) bool {
	h, p, err := net.SplitHostPort(s)
	if err != nil || h == "" {
		return false
	}

	n, err := strconv.Atoi(p)
	if err != nil || n < 1 || n > 65535 {
		return false
	}

	if ip := net.ParseIP(h); ip != nil {
		return !ip.IsUnspecified()
	}

	if len(h) > 253 {
		return false
	}

	for _, label := range strings.Split(h, ".") {
		if len(label) == 0 || len(label) > 63 ||
			label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}

		for _, r := range label {
			if !(r >= 'a' && r <= 'z') &&
				!(r >= 'A' && r <= 'Z') &&
				!(r >= '0' && r <= '9') &&
				r != '-' {
				return false
			}
		}
	}

	return true
}

// ValidHTTPSPeerAddress validates an advertised secure peer endpoint.
func ValidHTTPSPeerAddress(address string) bool {
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return false
	}

	if u.User != nil || u.RawQuery != "" || u.Fragment != "" ||
		(u.Path != "" && u.Path != "/") {
		return false
	}

	return ValidAdvertisedAddress(u.Host)
}

// Validate checks all configuration before application startup.
func (c Config) Validate() error {
	if !ValidID(c.ID) {
		return fmt.Errorf("invalid node ID %q: use 1-32 letters, numbers, underscores or hyphens", c.ID)
	}

	// Wildcard listeners are permitted only in explicit container mode.
	validListener := ValidAddress
	if c.ContainerMode {
		validListener = ValidListenAddress
	}

	if !validListener(c.HTTP) {
		return fmt.Errorf("invalid HTTP listening address %q", c.HTTP)
	}

	if c.Timeout <= 0 || c.Timeout > 30*time.Second {
		return fmt.Errorf("invalid timeout %s: must be greater than 0 and no more than 30 seconds", c.Timeout)
	}

	for name, address := range map[string]string{
		"TCP":  c.TCP,
		"UDP":  c.UDP,
		"gRPC": c.GRPC,
	} {
		if address != "" && !validListener(address) {
			return fmt.Errorf("invalid %s listening address %q", name, address)
		}
	}

	for name, address := range map[string]string{
		"HTTP advertised": c.HTTPAdvertise,
		"gRPC advertised": c.GRPCAdvertise,
	} {
		if address != "" && !ValidAdvertisedAddress(address) {
			return fmt.Errorf("invalid %s address %q", name, address)
		}
	}

	if strings.TrimSpace(c.DataDir) == "" {
		return fmt.Errorf("storage directory cannot be empty")
	}

	if strings.TrimSpace(c.SecurityMode) == "required" {
		if c.PeerHTTPAddr == "" || !validListener(c.PeerHTTPAddr) {
			return fmt.Errorf("invalid secure peer HTTP listening address %q", c.PeerHTTPAddr)
		}

		if !ValidHTTPSPeerAddress(c.PeerHTTPAdvertise) {
			return fmt.Errorf("invalid secure peer HTTPS advertised address %q", c.PeerHTTPAdvertise)
		}

		for _, seed := range c.PeerSeeds {
			if !ValidHTTPSPeerAddress(seed) {
				return fmt.Errorf("invalid secure peer seed address %q", seed)
			}
		}
	} else {
		if c.PeerHTTPAddr != "" && !validListener(c.PeerHTTPAddr) {
			return fmt.Errorf("invalid peer HTTP listening address %q", c.PeerHTTPAddr)
		}

		if c.PeerHTTPAdvertise != "" && !ValidHTTPSPeerAddress(c.PeerHTTPAdvertise) {
			return fmt.Errorf("invalid peer HTTPS advertised address %q", c.PeerHTTPAdvertise)
		}

		for _, seed := range c.PeerSeeds {
			if !ValidAdvertisedAddress(seed) {
				return fmt.Errorf("invalid peer seed address %q", seed)
			}
		}
	}

	return nil
}
