// A separate worked example, not an implementation of an assessed handler.
package main

import (
	"errors"
	"fmt"
	"strings"
)

func label(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", errors.New("label is empty")
	}
	return "lab-" + s, nil
}
func main() {
	s, e := label(" demo ")
	if e != nil {
		fmt.Println(e)
		return
	}
	fmt.Println(s)
}
