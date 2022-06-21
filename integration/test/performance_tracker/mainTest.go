//go:build linux && integration
// +build linux,integration
package main

import (
	"os"
	"testing"
)

func TestPerformanceTracker(t * testing.T){
	t.Log("Hello World")
	t.Log("Hash", os.Getenv("SHA"))
}