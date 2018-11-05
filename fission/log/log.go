/*
Copyright 2016 The Fission Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	os.Stderr.WriteString(fmt.Sprintf("[FATAL] %v\n", msg))
	os.Exit(1)
}

func Warn(msg interface{}) {
	os.Stderr.WriteString(fmt.Sprintf("[WARNING] %v\n", msg))
}

func Info(msg interface{}) {
	os.Stderr.WriteString(fmt.Sprintf("%v\n", msg))
}

func Verbose(verbosityLevel int, format string, args ...interface{}) {
	if Verbosity >= verbosityLevel {
		fmt.Printf(format+"\n", args...)
	}
}
