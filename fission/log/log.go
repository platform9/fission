package log

import (
	"fmt"
	"os"
)

var (
	// global Verbosity of our CLI
	Verbosity int
)

func Fatal(msg interface{}) {
	os.Stderr.WriteString(fmt.Sprintf("%v\n", msg))
	os.Exit(1)
}

func Warn(msg interface{}) {
	os.Stderr.WriteString(fmt.Sprintf("%v\n", msg))
}

func Verbose(verbosityLevel int, format string, args ...interface{}) {
	if Verbosity >= verbosityLevel {
		fmt.Printf(format+"\n", args...)
	}
}
