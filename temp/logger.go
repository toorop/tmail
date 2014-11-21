package main

import (
	"io"
	"log"
	"os"
)

// (from revel Thanks @robfig)
// Create a logger using log.* directives in app.conf plus the current settings
// on the default logger.
func getLogger(name string) *log.Logger {
	var logger *log.Logger

	// Create a logger with the requested output. (default to stderr)
	output := Config.StringDefault("log."+name+".output", "stderr")

	switch output {
	case "stdout":
		logger = newLogger(os.Stdout)
	case "stderr":
		logger = newLogger(os.Stderr)
	default:
		if output == "off" {
			output = os.DevNull
		}
		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalln("Failed to open log file", output, ":", err)
		}
		logger = newLogger(file)
	}

	// Set the prefix / flags.
	flags, found := Config.Int("log." + name + ".flags")
	if found {
		logger.SetFlags(flags)
	} else if name == "trace" {
		logger.SetFlags(TRACE.Flags())
	}

	prefix, found := Config.String("log." + name + ".prefix")
	if found {
		logger.SetPrefix(prefix)
	} else if name == "trace" {
		logger.SetPrefix(TRACE.Prefix())
	}

	return logger
}
func newLogger(wr io.Writer) *log.Logger {
	return log.New(wr, "", INFO.Flags())
}
