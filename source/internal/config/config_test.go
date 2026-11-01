package config

import (
	"io"
	"testing"
	"time"
)

func TestSmoke_ParseHelp(t *testing.T) {
	if _, e := Parse([]string{"-help"}, io.Discard); e == nil {
		t.Fatal("help must return flag.ErrHelp")
	}
}
func TestWeek01_Config(t *testing.T) {
	good := Config{ID: "node-a", HTTP: "127.0.0.1:8081", Timeout: 2 * time.Second}
	if e := good.Validate(); e != nil {
		t.Fatalf("valid configuration rejected: %v", e)
	}
	cases := []Config{good, good, good, good, good}
	cases[0].ID = ""
	cases[1].HTTP = ":8081"
	cases[2].HTTP = "127.0.0.1:0"
	cases[3].Timeout = 0
	cases[4].HTTP = "0.0.0.0:8081"
	for i, c := range cases {
		if c.Validate() == nil {
			t.Errorf("invalid case %d accepted", i)
		}
	}
}
func TestWeek01_UnknownFlag(t *testing.T) {
	if _, e := Parse([]string{"-bogus"}, io.Discard); e == nil {
		t.Fatal("unknown flag accepted")
	}
}

func TestWeek01_ConfigRejectsInvalidOptionalAddress(t *testing.T) {
	base := Config{
		ID:      "node-a",
		HTTP:    "127.0.0.1:8081",
		Timeout: 2 * time.Second,
	}

	cases := []struct {
		name   string
		change func(*Config)
	}{
		{
			name: "invalid TCP address",
			change: func(c *Config) {
				c.TCP = "0.0.0.0:9091"
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
