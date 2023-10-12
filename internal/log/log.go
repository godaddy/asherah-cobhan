package output

import (
	"fmt"
	"os"
)

var DebugLog func(interface{}) = nil
var DebugLogf func(format string, args ...interface{}) = nil
var ErrorLog func(interface{}) = stderrDebugLog
var ErrorLogf func(format string, args ...interface{}) = stderrDebugLogf

func EnableVerboseLog(flag bool) {
	if flag {
		DebugLog = stderrDebugLog
		DebugLogf = stderrDebugLogf
		DebugLog("asherah-cobhan: Enabled debug log")
	} else {
		DebugLog = nullDebugLog
		DebugLogf = nullDebugLogf
	}
}

func stderrDebugLog(output interface{}) {
	fmt.Fprintf(os.Stderr, "asherah-cobhan: %#v\n", output)
}

func stderrDebugLogf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "asherah-cobhan: "+format+"\n", args...)
}

func nullDebugLog(output interface{}) {
}

func nullDebugLogf(format string, args ...interface{}) {
}
