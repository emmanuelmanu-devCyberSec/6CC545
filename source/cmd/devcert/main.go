// devcert creates short-lived certificates and one JWT key for a CLOSED lab.
// It refuses to overwrite an existing directory. No generated secret is shipped.
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

func main() {
	out := flag.String("out", "../secrets", "new private output directory")
	flag.Parse()
	if e := generate(*out); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	fmt.Println("Created closed-lab certificates for node-a, node-b, node-c and JWT key. Keep the parent directory private; never commit it.")
}
func serial() (*big.Int, error) {
	n, e := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if e != nil {
		return nil, e
	}
	if n.Sign() == 0 {
		n.SetInt64(1)
	}
	return n, nil
}
func writePEM(path, kind string, b []byte, mode os.FileMode) error {
	return os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: kind, Bytes: b}), mode)
}
func generate(out string) error {
	if _, e := os.Lstat(out); e == nil {
		return fmt.Errorf("refusing to overwrite %s; choose a new directory", out)
	} else if !os.IsNotExist(e) {
		return e
	}
	if e := os.Mkdir(out, 0700); e != nil {
		return e
	}
	if e := os.Chmod(out, 0700); e != nil {
		return e
	}
	key, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		return e
	}
	n, e := serial()
	if e != nil {
		return e
	}
	now := time.Now()
	ca := &x509.Certificate{SerialNumber: n, Subject: pkix.Name{CommonName: "CW2 local teaching CA"}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(7 * 24 * time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign}
	raw, e := x509.CreateCertificate(rand.Reader, ca, ca, &key.PublicKey, key)
	if e != nil {
		return e
	}
	if e = writePEM(filepath.Join(out, "ca.crt"), "CERTIFICATE", raw, 0644); e != nil {
		return e
	}
	kb, e := x509.MarshalECPrivateKey(key)
	if e != nil {
		return e
	}
	if e = writePEM(filepath.Join(out, "ca.key"), "EC PRIVATE KEY", kb, 0600); e != nil {
		return e
	}
	for _, id := range []string{"node-a", "node-b", "node-c"} {
		dir := filepath.Join(out, id)
		if e = os.Mkdir(dir, 0755); e != nil {
			return e
		}
		nodeKey, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if e != nil {
			return e
		}
		n, e := serial()
		if e != nil {
			return e
		}
		uri, _ := url.Parse("spiffe://cw2.local/" + id)
		c := &x509.Certificate{SerialNumber: n, Subject: pkix.Name{CommonName: id}, DNSNames: []string{id, "localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, URIs: []*url.URL{uri}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(48 * time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}}
		der, e := x509.CreateCertificate(rand.Reader, c, ca, &nodeKey.PublicKey, key)
		if e != nil {
			return e
		}
		if e = writePEM(filepath.Join(dir, "server.crt"), "CERTIFICATE", der, 0644); e != nil {
			return e
		}
		b, e := x509.MarshalECPrivateKey(nodeKey)
		if e != nil {
			return e
		}
		// Deliberately readable inside a read-only Docker secret mount; the OUTER
		// secrets directory is 0700 and limits access on the host. CA key stays 0600.
		if e = writePEM(filepath.Join(dir, "server.key"), "EC PRIVATE KEY", b, 0644); e != nil {
			return e
		}
	}
	secret := make([]byte, 32)
	if _, e = rand.Read(secret); e != nil {
		return e
	}
	return os.WriteFile(filepath.Join(out, "jwt.key"), secret, 0644)
}
