//go:build ignore
// +build ignore

package main

import "C"

// Warmup initializes the Go runtime for JavaScript runtime compatibility.
// This function is called via FFI from JavaScript runtimes (like Bun) before 
// loading CGO-based N-API modules. It prevents runtime errors that occur when 
// crypto/x509 initialization runs in certain JavaScript runtime environments.
//
//export Warmup
func Warmup() C.int {
	return 1
}

func main() {}