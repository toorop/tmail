package rest

import (
	"bytes"
	"github.com/codegangsta/negroni"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/toorop/tmail/config"
	"github.com/toorop/tmail/logger"
	"github.com/toorop/tmail/scope"
)

func Test_Logger_init(t *testing.T) {
	var err error
	scope.Cfg = new(config.Config)
	scope.Log, err = logger.New(ioutil.Discard, false)
	assert.NoError(t, err)
}

func Test_Logger(t *testing.T) {
	var err error
	assert := assert.New(t)
	buff := bytes.NewBufferString("")
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "http://localhost/foobar", nil)
	assert.NoError(err)
	scope.Log, err = logger.New(buff, false)
	assert.NoError(err)

	l := NewLogger()
	n := negroni.New()
	// replace log for testing
	n.Use(l)
	n.UseHandler(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
	}))

	n.ServeHTTP(w, r)
	assert.Equal(http.StatusNotFound, w.Code)
	assert.False(len(buff.Bytes()) == 0)
}
