//go:build linux && integration
// +build linux,integration
package main

import (
	"os"
	"testing"
)

func TestPerformanceTracker(t * testing.T){
	t.Log("Hello World")
	t.log("Hash", os.Getenv("SHA"))
}