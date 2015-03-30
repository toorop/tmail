package rest

import (
	"github.com/toorop/tmail/scope"
	"net/http"
	"strings"
)

// log helper for INFO log
func logInfo(r *http.Request, msg ...string) {
	scope.Log.Info("http", r.RemoteAddr, "-", r.Method, r.RequestURI, "-", strings.Join(msg, " "))
}

// logError is a log helper for ERROR logs
func logError(r *http.Request, msg ...string) {
	scope.Log.Error("http", r.RemoteAddr, "-", r.Method, r.RequestURI, "-", strings.Join(msg, " "))
}

// logDebug is a log helper for Debug logs
func logDebug(r *http.Request, msg ...string) {
	scope.Log.Debug("http", r.RemoteAddr, "-", r.Method, r.RequestURI, "-", strings.Join(msg, " "))
}
