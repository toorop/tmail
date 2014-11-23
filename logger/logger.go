package logger

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
)

// Simple logger package to log to stdout

type Logger struct {
	debugEnabled bool
	debug        *log.Logger
	info         *log.Logger
	err          *log.Logger
	trace        *log.Logger
}

func New(debugEnabled bool) *Logger {
	return &Logger{
		debugEnabled: debugEnabled,
		debug:        log.New(os.Stdout, "", log.Ldate|log.Ltime),
		info:         log.New(os.Stdout, "", log.Ldate|log.Ltime),
		err:          log.New(os.Stdout, "", log.Ldate|log.Ltime),
		trace:        log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (l *Logger) Debug(v ...interface{}) {
	if !l.debugEnabled {
		return
	}
	msg := "DEBUG -"
	for i := range v {
		msg = fmt.Sprintf("%s %v", msg, v[i])
	}
	l.debug.Println(msg)
}

func (l *Logger) Info(v ...interface{}) {
	msg := "INFO -"
	for i := range v {
		msg = fmt.Sprintf("%s %v", msg, v[i])
	}
	l.info.Println(msg)
}

func (l *Logger) Error(v ...interface{}) {
	msg := "ERROR -"
	for i := range v {
		msg = fmt.Sprintf("%s %v", msg, v[i])
	}
	l.err.Println(msg)
}

func (l *Logger) Trace(v ...interface{}) {
	stack := debug.Stack()
	msg := "TRACE -"
	for i := range v {
		msg = fmt.Sprintf("%s %v", msg, v[i])
	}
	msg += "\r\nStack: \r\n" + fmt.Sprintf("%s", stack)
	l.trace.Println(msg)
}
