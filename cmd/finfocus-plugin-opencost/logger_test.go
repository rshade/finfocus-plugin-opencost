package main

import (
	"os"
	"strings"
	"testing"
)

func TestMainSetsServerLogger(t *testing.T) {
	body, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "plugin.SetLogger(logger)") {
		t.Fatal("run does not install the server logger")
	}
}
