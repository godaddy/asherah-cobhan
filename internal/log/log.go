package output

import (
	"fmt"
	"os"
)

var VerboseLog func(interface{}) = nil
var VerboseLogf func(format string, args ...interface{}) = nil

func EnableVerboseLog(flag bool) {
	if flag {
		VerboseLog = StderrDebugLog
		VerboseLogf = StderrDebugLogf
		VerboseLog("asherah-cobhan: Enabled debug log")
	} else {
		VerboseLog = NullDebugLog
		VerboseLogf = NullDebugLogf
	}
}

func StderrDebugLog(output interface{}) {
	fmt.Fprintf(os.Stderr, "asherah-cobhan: %#v\n", output)
}

func StderrDebugLogf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "asherah-cobhan:"+format+"\n", args...)
}

func NullDebugLog(output interface{}) {
}

func NullDebugLogf(format string, args ...interface{}) {
}
