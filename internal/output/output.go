package output

import (
	"fmt"
	"os"
)

var VerboseOutput func(interface{}) = nil
var VerboseOutputf func(format string, args ...interface{}) = nil

func EnableVerboseOutput(flag bool) {
	if flag {
		VerboseOutput = StderrDebugOutput
		VerboseOutputf = StderrDebugOutputf
		VerboseOutput("asherah-cobhan: Enabled debug output")
	} else {
		VerboseOutput = NullDebugOutput
		VerboseOutputf = NullDebugOutputf
	}
}

func StderrDebugOutput(output interface{}) {
	fmt.Fprintf(os.Stderr, "asherah-cobhan: %#v\n", output)
}

func StderrDebugOutputf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "asherah-cobhan:"+format+"\n", args...)
}

func NullDebugOutput(output interface{}) {
}

func NullDebugOutputf(format string, args ...interface{}) {
}
