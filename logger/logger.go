package logger

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
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

func New(logPath string, debugEnabled bool) (*Logger, error) {
	var err error
	var out io.Writer
	if logPath == "stdout" {
		out = os.Stdout
	} else if logPath == "discard" {
		out = ioutil.Discard
	} else {
		file := path.Join(logPath, "current.log")
		out, err = os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
	}

	hostname, _ := os.Hostname()
	return &Logger{
		debugEnabled: debugEnabled,
		debug:        log.New(out, "["+hostname+" - 127.0.0.1] ", log.Ldate|log.Ltime|log.Lmicroseconds),
		info:         log.New(out, "["+hostname+" - 127.0.0.1] ", log.Ldate|log.Ltime|log.Lmicroseconds),
		err:          log.New(out, "["+hostname+" - 127.0.0.1] ", log.Ldate|log.Ltime|log.Lmicroseconds),
		trace:        log.New(out, "["+hostname+" - 127.0.0.1] ", log.Ldate|log.Ltime|log.Lshortfile),
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

// gorm insterface
func (l *Logger) Print(v ...interface{}) {
	l.Debug(v)
}
