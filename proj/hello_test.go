package main

import (
	"testing"
)

func TestHello(t *testing.T) {
	if 1+2 != 3 {
		t.Fatal("I can't count")
	}
}
