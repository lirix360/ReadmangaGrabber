package logger

import (
	"log"
	"os"

	"github.com/kpango/glg"
)

// Log - ...
var Log *glg.Glg

func init() {
	var err error

	if _, err = os.Stat("grabber_log.log"); err == nil {
		err = os.Remove("grabber_log.log")
		if err != nil {
			log.Fatal("Ошибка при удалении старого лог-файла:", err)
		}
	}

	logFile := glg.FileWriter("grabber_log.log", 0644)

	Log = glg.Get().
		SetMode(glg.BOTH).
		AddLevelWriter(glg.INFO, logFile).
		AddLevelWriter(glg.ERR, logFile).
		AddLevelWriter(glg.FATAL, logFile)
}
