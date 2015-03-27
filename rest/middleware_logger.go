package rest

import (
	"fmt"
	"github.com/codegangsta/negroni"
	"github.com/toorop/tmail/logger"
	"github.com/toorop/tmail/scope"
	//"log"
	"net/http"
	"time"
)

type Logger struct {
	// Logger inherits from log.Logger used to log messages with the Logger middleware
	*logger.Logger
}

// NewLogger returns a new Logger instance
func NewLogger() *Logger {
	return &Logger{scope.Log}
}

//
func (l *Logger) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	start := time.Now()
	//l.Info("http started", r.Method, r.URL.Path)
	next(rw, r)
	res := rw.(negroni.ResponseWriter)
	l.Info(fmt.Sprintf("http %s %s %s %v %s %v", r.RemoteAddr, r.Method, r.URL.Path, res.Status(), http.StatusText(res.Status()), time.Since(start)))
}
