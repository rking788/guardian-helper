package main

import (
	"os"

	"github.com/kpango/glg"
)

// log level constants to determine which labels to enable and which to disable
const (
	all uint = iota
	debug
	info
	warning
	err
)

var infolog *os.File

// ConfigureLogging will setup the glg logging package with the correct file destination
// coloring, etc. as desired for the entire application.
func ConfigureLogging(level string, logPath string) {

	if logPath != "" {
		infolog = glg.FileWriter(logPath, 0644)
		glg.Get().AddWriter(infolog)
	}

	glg.Get().
		SetMode(glg.BOTH).
		EnableColor().
		SetLevelMode(glg.LOG, glg.NONE)

	// Map the config log level value to an internal representation that is easier
	// to perform equality operations on.
	desiredLevel := map[string]uint{
		"all":     all,
		"debug":   debug,
		"info":    info,
		"warning": warning,
		"error":   err,
	}[level]

	for _, glgLevel := range []string{glg.DEBG, glg.INFO, glg.WARN, glg.ERR} {
		glg.Get().SetLevelMode(glgLevel, glgDestination(glgLevel, desiredLevel))
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

func glgDestination(glgLevel string, desiredLevel uint) int {

	// Special case where we want ALL logging.
	if desiredLevel == all {
		return glg.BOTH
	}

	enabled := false
	switch glgLevel {
	case glg.DEBG:
		enabled = desiredLevel <= debug
	case glg.INFO:
		enabled = desiredLevel <= info
	case glg.WARN:
		enabled = desiredLevel <= warning
	case glg.ERR:
		enabled = desiredLevel <= err
	}

	// NOTE: Currently does not support sending log to ONLY the Writer,
	// needs to be all or nothing right now
	if enabled {
		return glg.BOTH
	}
	return glg.NONE
}

// CloseLogger is responsible for closing any resources used for logging.
func CloseLogger() {
	if infolog != nil {
		infolog.Close()
	}
}
