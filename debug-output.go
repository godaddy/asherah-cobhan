package main

import "fmt"

func StdoutDebugOutput(output interface{}) {
	fmt.Printf("%#v\n", output)
}

func NullDebugOutput(output interface{}) {
}
