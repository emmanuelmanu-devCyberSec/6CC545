// probe is supplied health-check infrastructure. It verifies TLS when used.
package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	u := flag.String("url", "http://127.0.0.1:8080/healthz", "health URL")
	ca := flag.String("ca", "", "PEM trust root")
	flag.Parse()
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if *ca != "" {
		pem, e := os.ReadFile(*ca)
		if e != nil {
			fail(e)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			fail(fmt.Errorf("invalid CA PEM"))
		}
		cfg.RootCAs = pool
	}
	c := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{TLSClientConfig: cfg}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	r, e := c.Get(*u)
	if e != nil {
		fail(e)
	}
	defer r.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(r.Body, 4096))
	if r.StatusCode != http.StatusOK {
		fail(fmt.Errorf("health status %d", r.StatusCode))
	}
}
func fail(e error) { fmt.Fprintln(os.Stderr, e); os.Exit(1) }
