package rest

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/toorop/tmail/config"
	"github.com/toorop/tmail/logger"
	"github.com/toorop/tmail/scope"
)

func Test_authorized(t *testing.T) {
	var err error
	assert := assert.New(t)
	scope.Cfg = new(config.Config)
	scope.Log, err = logger.New("discard", false)
	assert.NoError(err)
	scope.Cfg.SetRestServerLogin("good")
	scope.Cfg.SetRestServerPasswd("good")

	// no Auth
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "http://localhost/foobar", nil)
	assert.False(authorized(w, r))
	assert.Equal(w.Code, http.StatusUnauthorized)
	assert.Equal(w.Header().Get("WWW-Authenticate"), "Basic realm=tmail REST server")

	// bad auth
	r.SetBasicAuth("bad", "bad")
	assert.False(authorized(w, r))
	r.SetBasicAuth("good", "bad")
	assert.False(authorized(w, r))
	r.SetBasicAuth("bad", "good")
	assert.False(authorized(w, r))
	assert.Equal(w.Code, http.StatusUnauthorized)

	// good auth
	w = httptest.NewRecorder()
	r.SetBasicAuth("good", "good")
	assert.True(authorized(w, r))
	assert.Equal(w.Code, 200)
}
