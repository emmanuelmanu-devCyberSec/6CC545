// devtoken is a supplied local demonstration issuer, not a login service.
package main

import (
	"flag"
	"fmt"
	jwt "github.com/golang-jwt/jwt/v5"
	"os"
	"time"
)

func main() {
	key := flag.String("key", "../secrets/jwt.key", "random key file")
	role := flag.String("role", "reader", "reader or writer")
	sub := flag.String("sub", "student", "test subject")
	ttl := flag.Duration("ttl", 5*time.Minute, "1s..15m")
	iss := flag.String("iss", "cw2-lab", "issuer")
	aud := flag.String("aud", "cw2-api", "audience")
	flag.Parse()
	b, e := os.ReadFile(*key)
	if e != nil {
		fail(e)
	}
	if len(b) < 32 || (*role != "reader" && *role != "writer") || *sub == "" || *iss == "" || *aud == "" || *ttl < time.Second || *ttl > 15*time.Minute {
		fail(fmt.Errorf("invalid key, claims or lifetime"))
	}
	now := time.Now()
	c := jwt.MapClaims{"sub": *sub, "role": *role, "iss": *iss, "aud": *aud, "iat": now.Unix(), "nbf": now.Add(-time.Second).Unix(), "exp": now.Add(*ttl).Unix()}
	s, e := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(b)
	if e != nil {
		fail(e)
	}
	fmt.Println(s)
}
func fail(e error) { fmt.Fprintln(os.Stderr, e); os.Exit(1) }
