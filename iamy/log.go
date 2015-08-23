package iamy

import (
	"fmt"
	"strings"
)

type LoggerFunc func(string)

var noLog = func(string) {}

var Logger LoggerFunc = noLog

func verboseLog(s ...string) {
	Logger(strings.Join(s, " "))
}

func verboseLogf(format string, a ...interface{}) {
	verboseLog(fmt.Sprintf(format, a...))
}
