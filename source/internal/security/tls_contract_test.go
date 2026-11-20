//go:build acceptance

package security

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fixtures(t *testing.T) (string, string, string) {
	t.Helper()
	d := t.TempDir()
	k, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	ca := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test CA"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	der, e := x509.CreateCertificate(rand.Reader, ca, ca, &k.PublicKey, k)
	if e != nil {
		t.Fatal(e)
	}
	cap := filepath.Join(d, "ca.crt")
	if e = os.WriteFile(cap, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600); e != nil {
		t.Fatal(e)
	}
	nk, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	leaf := &x509.Certificate{SerialNumber: big.NewInt(2), DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}}
	der, e = x509.CreateCertificate(rand.Reader, leaf, ca, &nk.PublicKey, k)
	if e != nil {
		t.Fatal(e)
	}
	cp := filepath.Join(d, "node.crt")
	kp := filepath.Join(d, "node.key")
	_ = os.WriteFile(cp, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600)
	b, e := x509.MarshalECPrivateKey(nk)
	if e != nil {
		t.Fatal(e)
	}
	_ = os.WriteFile(kp, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b}), 0600)
	return cp, kp, cap
}
func TestMutualTLSIdentityAndNegativePaths(t *testing.T) {
	cert, key, ca := fixtures(t)
	sc, e := ServerTLS(cert, key, ca, true)
	if e != nil {
		t.Fatal(e)
	}
	if sc.ClientAuth != tls.RequireAndVerifyClientCert || sc.MinVersion < tls.VersionTLS12 {
		t.Fatal("server policy")
	}
	s := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok") }))
	s.TLS = sc
	s.StartTLS()
	defer s.Close()
	cc, e := ClientTLS(cert, key, ca, "localhost")
	if e != nil {
		t.Fatal(e)
	}
	if cc.InsecureSkipVerify || cc.MinVersion < tls.VersionTLS12 {
		t.Fatal("client policy")
	}
	call := func(c *tls.Config) error {
		tr := &http.Transport{TLSClientConfig: c}
		defer tr.CloseIdleConnections()
		cl := &http.Client{Transport: tr, Timeout: 2 * time.Second}
		r, e := cl.Get(s.URL)
		if e != nil {
			return e
		}
		defer r.Body.Close()
		_, e = io.Copy(io.Discard, r.Body)
		return e
	}
	if e = call(cc); e != nil {
		t.Fatalf("valid mTLS %v", e)
	}
	bad := cc.Clone()
	bad.ServerName = "wrong-node"
	if e = call(bad); e == nil {
		t.Fatal("wrong SAN accepted")
	}
	bad = cc.Clone()
	bad.RootCAs = x509.NewCertPool()
	if e = call(bad); e == nil {
		t.Fatal("untrusted issuer accepted")
	}
	bad = cc.Clone()
	bad.Certificates = nil
	if e = call(bad); e == nil {
		t.Fatal("missing client certificate accepted")
	}
	if _, e = ServerTLS("missing", key, ca, true); e == nil {
		t.Fatal("missing cert accepted")
	}
	if _, e = ClientTLS(cert, key, ca, ""); e == nil {
		t.Fatal("missing identity accepted")
	}
}
