package rest

import (
	"net/http"
	"strings"

	"github.com/toorop/tmail/core"
)

// log helper for INFO log
func logInfo(r *http.Request, msg ...string) {
	core.Log.Info("http", r.RemoteAddr, "-", r.Method, r.RequestURI, "-", strings.Join(msg, " "))
}

// logError is a log helper for ERROR logs
func logError(r *http.Request, msg ...string) {
	core.Log.Error("http", r.RemoteAddr, "-", r.Method, r.RequestURI, "-", strings.Join(msg, " "))
}

// logDebug is a log helper for Debug logs
func logDebug(r *http.Request, msg ...string) {
	core.Log.Debug("http", r.RemoteAddr, "-", r.Method, r.RequestURI, "-", strings.Join(msg, " "))
}
