package logger

import (
	"fmt"
	"io"
	//"io/ioutil"
	"log"
	"os"
	//"path"
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

func New(out io.Writer, debugEnabled bool) (*Logger, error) {
	hostname, _ := os.Hostname()
	return &Logger{
		debugEnabled: debugEnabled,
		debug:        log.New(out, "["+hostname+"] ", log.Ldate|log.Ltime|log.Lmicroseconds),
		info:         log.New(out, "["+hostname+"] ", log.Ldate|log.Ltime|log.Lmicroseconds),
		err:          log.New(out, "["+hostname+"] ", log.Ldate|log.Ltime|log.Lmicroseconds),
		trace:        log.New(out, "["+hostname+"] ", log.Ldate|log.Ltime|log.Lshortfile),
	}, nil
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

// nsq interface
func (l *Logger) Output(calldepth int, s string) error {
	l.Debug(s)
	return nil
}

// gorm interface
func (l *Logger) Print(v ...interface{}) {
	l.Debug(v)
}
