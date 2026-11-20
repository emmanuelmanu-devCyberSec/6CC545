package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateAndRefuseOverwrite(t *testing.T) {
	d := filepath.Join(t.TempDir(), "secrets")
	if e := generate(d); e != nil {
		t.Fatal(e)
	}
	caPEM, e := os.ReadFile(filepath.Join(d, "ca.crt"))
	if e != nil {
		t.Fatal(e)
	}
	p := x509.NewCertPool()
	if !p.AppendCertsFromPEM(caPEM) {
		t.Fatal("bad CA")
	}
	for _, id := range []string{"node-a", "node-b", "node-c"} {
		pair, e := tls.LoadX509KeyPair(filepath.Join(d, id, "server.crt"), filepath.Join(d, id, "server.key"))
		if e != nil {
			t.Fatal(e)
		}
		c, e := x509.ParseCertificate(pair.Certificate[0])
		if e != nil {
			t.Fatal(e)
		}
		if _, e = c.Verify(x509.VerifyOptions{DNSName: id, Roots: p}); e != nil {
			t.Fatal(e)
		}
		if len(c.URIs) != 1 || c.URIs[0].String() != "spiffe://cw2.local/"+id {
			t.Fatal("wrong peer URI")
		}
	}
	b, _ := os.ReadFile(filepath.Join(d, "jwt.key"))
	if len(b) != 32 {
		t.Fatal("JWT key length")
	}
	block, _ := pem.Decode(caPEM)
	if block == nil {
		t.Fatal("PEM")
	}
	if e := generate(d); e == nil {
		t.Fatal("overwrote keys")
	}
}
