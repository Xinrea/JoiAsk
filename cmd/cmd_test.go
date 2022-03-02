package main

import (
	"strings"
	"testing"
)

func TestSplit(t *testing.T) {
	t.Log(len(strings.Split(",", ",")))
	t.Fail()
}
