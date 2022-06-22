//go:build linux && integration
// +build linux,integration
package main

import (
	"fmt"
	"os"
	"testing"
)

func TestPerformanceTracker(t * testing.T){
	t.Log("Hello World")
	t.Log("Hash", os.Getenv("SHA"))
}


func main(){
	fmt.Println("Hello World")
	fmt.Println("Hash", os.Getenv("SHA"), "Commit Date:",os.Getenv("SHA_DATE"))
}