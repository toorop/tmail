package rest

import (
	"fmt"
	"github.com/codegangsta/negroni"
	"net/http"
	"time"
)

type Logger struct {
}

// NewLogger returns a new Logger instance
func NewLogger() *Logger {
	return &Logger{}
}

//
func (l *Logger) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	start := time.Now()
	next(rw, r)
	res := rw.(negroni.ResponseWriter)
	logInfo(r, fmt.Sprintf("%v %s %v", res.Status(), http.StatusText(res.Status()), time.Since(start)))
}
