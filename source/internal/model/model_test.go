package model

import (
	"strings"
	"testing"
)

func TestWeek03_MetadataValidation(t *testing.T) {
	good := FileMetadata{Name: "notes.txt", Size: 1, SHA256: strings.Repeat("0", 64), ModifiedUnix: 1}
	if e := good.Validate(); e != nil {
		t.Fatal(e)
	}
	cases := []FileMetadata{good, good, good, good}
	cases[0].Name = "../escape"
	cases[1].Size = -1
	cases[2].SHA256 = "not-a-hash"
	cases[3].ModifiedUnix = -1
	for _, f := range cases {
		if f.Validate() == nil {
			t.Fatal("invalid metadata accepted")
		}
	}
}
