package rest

import (
	"bytes"
	"encoding/base64"
	"github.com/toorop/tmail/scope"
	"net/http"
)

// ServeHTTP implementation of interface
func authorized(w http.ResponseWriter, r *http.Request) bool {
	// Headers Authorization found ?
	hAuth := r.Header.Get("authorization")
	if hAuth == "" {
		w.Header().Set("WWW-Authenticate", "Basic realm=tmail REST server")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return false
	}
	// check credential
	if hAuth[:5] != "Basic" {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(hAuth[6:])
	if err != nil {
		logError(r, "on decoding http auth credentials:", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return false
	}
	credentials := bytes.SplitN(decoded, []byte{58}, 2)

	if bytes.Compare([]byte(scope.Cfg.GetRestServerLogin()), credentials[0]) != 0 || bytes.Compare([]byte(scope.Cfg.GetRestServerPasswd()), credentials[1]) != 0 {
		logError(r, "bad authentification. Login:", string(credentials[0]), "password:", string(credentials[1]))
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return false
	}
	return true
}
