package logger

// Simple logger package to log to stdout

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	CRITICAL = 4
	ERROR    = 3
	WARNING  = 2
	NOTICE   = 1
	DEBUG    = 0
)

var levels = map[string]int{
	"critical": CRITICAL,
	"error":    ERROR,
	"warning":  WARNING,
	"notice":   NOTICE,
	"debug":    DEBUG,
}

var rlevels = [5]string{
	"debug",
	"notice",
	"warning",
	"error",
	"critical",
}

var lw *logWritter

// Logger define a logger
type Logger struct {
	level        int
	addTimestamp bool
}

// logWritter represent a writter
type logWritter struct {
	sync.Mutex
}

// writelog write msd to stdout
func (lw *logWritter) writeLog(msg string) {
	lw.Lock()
	fmt.Println(msg)
	lw.Unlock()
}

// init
func init() {
	lw = new(logWritter)
}

// SetLevel is used to set the defaul log level, if
// log are pushed under this level they are note published
func (l *Logger) SetLevel(level string) {
	level = strings.ToLower(level)
	l.level = levels[level]
}

// SetTimeStamp if used ti config the addTimeStamp option
func (l *Logger) SetTimeStamp(ts bool) {
	l.addTimestamp = ts
}

// log is the internal log method
func (l *Logger) log(level int, v ...interface{}) {
	msg := ""
	if level >= l.level {
		if l.addTimestamp {
			msg = fmt.Sprintf("%d - ", time.Now().UnixNano())
		}
		// level
		msg = fmt.Sprintf("%s%s - ", msg, strings.ToTitle(rlevels[level]))

		if len(v) == 1 {
			msg += fmt.Sprintf("%v", v)
		} else {
			for i := range v {
				msg = fmt.Sprintf("%s%v", msg, v[i])
			}
		}
		lw.WriteLog(msg)
	}
}

// Debug log at debug level
func (l *Logger) Debug(v ...interface{}) {
	l.log(DEBUG, v...)
}

// Notice log at notice level
func (l *Logger) Notice(v ...interface{}) {
	l.log(NOTICE, v...)
}

// Warning log at warning level
func (l *Logger) Warning(v ...interface{}) {
	l.log(WARNING, v...)
}

// Error log at error level
func (l *Logger) Error(v ...interface{}) {
	l.log(ERROR, v...)
}

// Critical log at critical level
func (l *Logger) Critical(v ...interface{}) {
	l.log(CRITICAL, v...)
}
