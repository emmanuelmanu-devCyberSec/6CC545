package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	url := flag.String("url", "http://127.0.0.1:8081/healthz", "target URL")
	method := flag.String("method", "GET", "HTTP method")
	file := flag.String("file", "", "request body file")
	media := flag.String("type", "application/json", "request media type")
	expect := flag.Int("expect", 200, "expected HTTP status; mismatch exits nonzero")
	flag.Parse()
	var b []byte
	var e error
	if *file != "" {
		b, e = os.ReadFile(*file)
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
	}
	r, e := http.NewRequest(*method, *url, bytes.NewReader(b))
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	if *file != "" {
		r.Header.Set("Content-Type", *media)
	}
	resp, e := (&http.Client{Timeout: 3 * time.Second}).Do(r)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	defer resp.Body.Close()
	data, e := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if e != nil || len(data) > 1<<20 {
		fmt.Fprintln(os.Stderr, "response unreadable or too large")
		os.Exit(1)
	}
	fmt.Printf("HTTP %d\nContent-Type: %s\n%s", resp.StatusCode, resp.Header.Get("Content-Type"), data)
	if resp.StatusCode != *expect {
		fmt.Fprintf(os.Stderr, "expected %d\n", *expect)
		os.Exit(1)
	}
}
