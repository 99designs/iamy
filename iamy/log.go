package iamy

import (
	"fmt"
	"strings"
)

type LoggerFunc func(string)

var noLog = func(string) {}

var Logger LoggerFunc = noLog

func logPrintln(s ...string) {
	Logger(strings.Join(s, " "))
}

func logPrintf(format string, a ...interface{}) {
	Logger(fmt.Sprintf(format, a...))
}
