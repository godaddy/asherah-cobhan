package debug_output

import "fmt"

func StdoutDebugOutput(output interface{}) {
	fmt.Printf("%#v\n", output)
}

func StdoutDebugOutputf(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func NullDebugOutput(output interface{}) {
}

func NullDebugOutputf(format string, args ...interface{}) {
}
