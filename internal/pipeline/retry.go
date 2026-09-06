package pipeline

import (
	"log"
	"os"
)

var warnLog = log.New(os.Stderr, "warn: ", 0)

func warnf(format string, args ...any) { warnLog.Printf(format, args...) }
