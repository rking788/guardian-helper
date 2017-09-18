package main

import (
	"os"

	"github.com/kpango/glg"
)

const (
	// TODO: This stuff should be passed in via a config object.
	// DebugLog indicates if debug logs should be included in the logging output. True if it should; False otherwise.
	DebugLog = true
	// LogFilename is the file where logs should be written on disk. This should be correctly configured with logrotate.
	LogFilename = "/var/log/warmind-network.log"
)

var infolog *os.File

// ConfigureLogging will setup the glg logging package with the correct file destination
// coloring, etc. as desired for the entire application.
func ConfigureLogging() {
	infolog = glg.FileWriter(LogFilename, 0644)

	glg.Get().
		SetMode(glg.BOTH).
		EnableColor().
		AddWriter(infolog).
		SetLevelMode(glg.LOG, glg.NONE)

	if DebugLog == false {
		// Disable debug and info logs if we don't want them.
		glg.Get().
			SetLevelMode(glg.DEBG, glg.NONE)
	}

	//glg.Info("info")
	// glg.Infof("%s : %s", "info", "formatted")
	// glg.Log("log")
	// glg.Logf("%s : %s", "info", "formatted")
	// glg.Debug("debug")
	// glg.Debugf("%s : %s", "info", "formatted")
	// glg.Warn("warn")
	// glg.Warnf("%s : %s", "info", "formatted")
	// glg.Error("error")
	// glg.Errorf("%s : %s", "info", "formatted")
	// glg.Success("ok")
	// glg.Successf("%s : %s", "info", "formatted")
	// glg.Fail("fail")
	// glg.Failf("%s : %s", "info", "formatted")
	// glg.Print("Print")
	// glg.Println("Println")
	// glg.Printf("%s : %s", "printf", "formatted")
}

func CloseLogger() {
	infolog.Close()
}
