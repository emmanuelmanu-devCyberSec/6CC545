package config

import (
	"errors"
	"flag"
	"io"
	"testing"
	"time"
)

// clearConfigEnv prevents existing machine environment variables
// from interfering with configuration tests.
func clearConfigEnv(t *testing.T) {
	t.Helper()

	variables := []string{
		"NODE_ID",
		"HTTP_ADDR",
		"TCP_ADDR",
		"UDP_ADDR",
		"GRPC_ADDR",
		"HTTP_ADVERTISE",
		"GRPC_ADVERTISE",
		"DATA_DIR",
		"PEER_SEEDS",
		"SEED",
		"NETWORK_TIMEOUT",
		"CONTAINER_MODE",
	}

	for _, name := range variables {
		t.Setenv(name, "")
	}
}

func TestSmoke_ParseHelp(t *testing.T) {
	clearConfigEnv(t)

	if _, err := Parse([]string{"-help"}, io.Discard); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got: %v", err)
	}
}

func TestWeek01_Config(t *testing.T) {
	good := Config{
		ID:      "node-a",
		HTTP:    "127.0.0.1:8081",
		DataDir: "data/node-a",
		Timeout: 2 * time.Second,
	}

	if err := good.Validate(); err != nil {
		t.Fatalf("valid configuration rejected: %v", err)
	}

	cases := []Config{good, good, good, good, good}
	cases[0].ID = ""
	cases[1].HTTP = ":8081"
	cases[2].HTTP = "127.0.0.1:0"
	cases[3].Timeout = 0
	cases[4].DataDir = ""

	for i, c := range cases {
		if c.Validate() == nil {
			t.Errorf("invalid case %d accepted", i)
		}
	}
}

func TestWeek01_UnknownFlag(t *testing.T) {
	clearConfigEnv(t)

	if _, err := Parse([]string{"-bogus"}, io.Discard); err == nil {
		t.Fatal("unknown flag accepted")
	}
}

func TestWeek01_ConfigRejectsInvalidOptionalAddress(t *testing.T) {
	base := Config{
		ID:      "node-a",
		HTTP:    "127.0.0.1:8081",
		DataDir: "data/node-a",
		Timeout: 2 * time.Second,
	}

	cases := []struct {
		name   string
		change func(*Config)
	}{
		{
			name: "invalid TCP address",
			change: func(c *Config) {
				c.TCP = "example.com:9091"
			},
		},
		{
			name: "invalid UDP address",
			change: func(c *Config) {
				c.UDP = "127.0.0.1:0"
			},
		},
		{
			name: "timeout exceeds 30 seconds",
			change: func(c *Config) {
				c.Timeout = 31 * time.Second
			},
		},
		{
			name: "invalid advertised HTTP address",
			change: func(c *Config) {
				c.HTTPAdvertise = "http://node-a:8081"
			},
		},
		{
			name: "invalid advertised gRPC address",
			change: func(c *Config) {
				c.GRPCAdvertise = "node-a:0"
			},
		},
		{
			name: "wildcard advertised HTTP address",
			change: func(c *Config) {
				c.HTTPAdvertise = "0.0.0.0:8081"
			},
		},
		{
			name: "wildcard advertised gRPC address",
			change: func(c *Config) {
				c.GRPCAdvertise = ":::9090"
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := base
			tc.change(&c)

			if err := c.Validate(); err == nil {
				t.Fatal("expected invalid configuration to be rejected")
			}
		})
	}
}

func TestConfigRejectsWildcardListenerInLocalMode(t *testing.T) {
	c := Config{
		ID:      "node-a",
		HTTP:    "0.0.0.0:8081",
		DataDir: "data/node-a",
		Timeout: 2 * time.Second,
	}

	if err := c.Validate(); err == nil {
		t.Fatal("wildcard listener should be rejected in local mode")
	}
}

func TestConfigAllowsWildcardListenerInContainerMode(t *testing.T) {
	c := Config{
		ID:            "node-a",
		HTTP:          "0.0.0.0:8081",
		GRPC:          "0.0.0.0:9090",
		DataDir:       "data/node-a",
		Timeout:       2 * time.Second,
		ContainerMode: true,
	}

	if err := c.Validate(); err != nil {
		t.Fatalf("wildcard listener should be accepted in container mode: %v", err)
	}
}

func TestParseCLIOverridesEnvironment(t *testing.T) {
	clearConfigEnv(t)

	t.Setenv("NODE_ID", "environment-node")
	t.Setenv("HTTP_ADDR", "127.0.0.1:8081")
	t.Setenv("NETWORK_TIMEOUT", "2s")

	c, err := Parse([]string{
		"-id", "cli-node",
		"-http", "127.0.0.1:8082",
		"-timeout", "3s",
	}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	if c.ID != "cli-node" {
		t.Errorf("CLI ID did not take precedence: got %q", c.ID)
	}
	if c.HTTP != "127.0.0.1:8082" {
		t.Errorf("CLI HTTP address did not take precedence: got %q", c.HTTP)
	}
	if c.Timeout != 3*time.Second {
		t.Errorf("CLI timeout did not take precedence: got %s", c.Timeout)
	}
}

func TestParseCLIOverridesInvalidEnvironmentTimeout(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("NETWORK_TIMEOUT", "not-a-duration")

	c, err := Parse([]string{"-timeout", "3s"}, io.Discard)
	if err != nil {
		t.Fatalf("CLI timeout should override invalid environment value: %v", err)
	}

	if c.Timeout != 3*time.Second {
		t.Fatalf("expected 3s timeout, got %s", c.Timeout)
	}
}

func TestParseRejectsInvalidEnvironmentTimeout(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("NETWORK_TIMEOUT", "not-a-duration")

	if _, err := Parse(nil, io.Discard); err == nil {
		t.Fatal("invalid environment timeout should be rejected")
	}
}

func TestParseContainerModeFromEnvironment(t *testing.T) {
	clearConfigEnv(t)

	t.Setenv("CONTAINER_MODE", "true")
	t.Setenv("HTTP_ADDR", "0.0.0.0:8080")
	t.Setenv("GRPC_ADDR", "0.0.0.0:9090")

	c, err := Parse(nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	if !c.ContainerMode {
		t.Fatal("container mode was not enabled")
	}
}

func TestParseContainerFlagOverridesEnvironment(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("CONTAINER_MODE", "false")

	c, err := Parse([]string{
		"-container",
		"-http", "0.0.0.0:8080",
	}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	if !c.ContainerMode {
		t.Fatal("container CLI flag should override environment")
	}
}

func TestParseRejectsInvalidContainerMode(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("CONTAINER_MODE", "sometimes")

	if _, err := Parse(nil, io.Discard); err == nil {
		t.Fatal("invalid CONTAINER_MODE should be rejected")
	}
}

func TestValidAdvertisedAddress(t *testing.T) {
	valid := []string{
		"node-a:8081",
		"peer.example.com:9090",
		"192.168.1.10:8081",
		"[::1]:9090",
	}

	for _, address := range valid {
		if !ValidAdvertisedAddress(address) {
			t.Errorf("expected advertised address %q to be valid", address)
		}
	}

	invalid := []string{
		"",
		"node-a:0",
		"http://node-a:8081",
		"bad host:8081",
		"0.0.0.0:8081",
		"[::]:9090",
	}

	for _, address := range invalid {
		if ValidAdvertisedAddress(address) {
			t.Errorf("expected advertised address %q to be invalid", address)
		}
	}
}
