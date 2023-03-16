package log

import (
	"log"
	"os"
	"io/ioutil"
	"go.uber.org/zap"
)

// Err stderr logger
var Err *log.Logger
// Out stdout logger
var Out *log.Logger
// Debug stdout verbose logger
var Debug *log.Logger
// Report stdout logger
var Report *log.Logger

// Zap logger
var Logger *zap.Logger

// InitLoggers sets logger
func InitLoggers(debugFlag bool) {
	Err = log.New(os.Stderr, "!!! ", log.LstdFlags)
	Out = log.New(os.Stdout, "    ", log.LstdFlags)
	if debugFlag {
		Debug = log.New(os.Stdout, "(d) ", log.LstdFlags)
	} else {
		Debug = log.New(ioutil.Discard, "(d) ", log.LstdFlags)
	}
	Report = log.New(os.Stdout, "+++ ", log.LstdFlags)

	Logger, _ = zap.NewProduction()
	defer Logger.Sync()
}
