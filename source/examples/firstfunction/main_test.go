package main

import "testing"

func TestSmoke_LabelExample(t *testing.T) {
	got, e := label(" demo ")
	if e != nil || got != "lab-demo" {
		t.Fatal(got, e)
	}
	if _, e = label(" "); e == nil {
		t.Fatal("want error")
	}
}
