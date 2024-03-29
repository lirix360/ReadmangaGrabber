package logger

import (
	"github.com/charmbracelet/log"
	"os"
	"time"
)

var Log *log.Logger

func init() {
	Log = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		TimeFormat:      time.TimeOnly,
		Prefix:          "🍪",
		ReportCaller:    true,
	})
}
