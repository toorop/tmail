package rest

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	//"net/http/httptest"
	"testing"

	"github.com/toorop/tmail/config"
	"github.com/toorop/tmail/logger"
	"github.com/toorop/tmail/scope"
)

func Test_init(t *testing.T) {
	var err error
	scope.Cfg = new(config.Config)
	scope.Log, err = logger.New("discard", false)
	assert.NoError(t, err)

}

func Test_log(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://localhost/foobar", nil)
	assert.NotPanics(t, func() { logDebug(r, "foo") })
	assert.NotPanics(t, func() { logInfo(r, "foo") })
	assert.NotPanics(t, func() { logError(r, "foo") })
}
