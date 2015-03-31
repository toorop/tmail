package rest

import (
	"github.com/toorop/tmail/scope"
	"io"
	"io/ioutil"
	"net/http"
)

// httpGetBody returns http body as string
func httpGetBody(r *http.Request) ([]byte, error) {
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, body_read_limit))
	if err != nil {
		return []byte{}, err
	}
	return body, r.Body.Close()
}

// httpErrorJson send and json formated error
func httpErrorJson(w http.ResponseWriter, httpStatus int, msg, raw string) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(httpStatus)
	w.Write([]byte(`{"message":"` + msg + `","raw":"` + raw + `"}`))
}

// httpGetScheme returns http ou https
func httpGetScheme() string {
	scheme := "http"
	if scope.Cfg.GetRestServerIsTls() {
		scheme = "https"
	}
	return scheme
}
