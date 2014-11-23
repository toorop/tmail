package logger

import (
	"fmt"
	"log"
	"os"
)

// Simple logger package to log to stdout
var (
	debug, info, err *log.Logger
	debugEnabled     bool
)

type logger struct {
	debugEnabled bool
	debug        *log.Logger
	info         *log.Logger
	err          *log.Logger
}

func New(debugEnabled bool) *logger {
	return &logger{
		debugEnabled: debugEnabled,
		debug:        log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile),
		info:         log.New(os.Stdout, "", log.Ldate|log.Ltime),
		err:          log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (l *logger) Debug(v ...interface{}) {
	if !l.debugEnabled {
		return
	}
	msg := "DEBUG -"
	for i := range v {
		msg = fmt.Sprintf("%s %v", msg, v[i])
	}
	l.debug.Println(msg)
}

func (l *logger) Info(v ...interface{}) {
	msg := "Info -"
	for i := range v {
		msg = fmt.Sprintf("%s %v", msg, v[i])
	}
	l.info.Println(msg)
}

func (l *logger) Error(v ...interface{}) {
	msg := "ERROR -"
	for i := range v {
		msg = fmt.Sprintf("%s %v", msg, v[i])
	}
	l.err.Println(msg)
}
