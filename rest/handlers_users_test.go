package rest

import (
	"bytes"
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"github.com/nbio/httpcontext"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/toorop/tmail/core"
	"github.com/toorop/tmail/logger"
	"github.com/toorop/tmail/scope"
)

func TestHandlerUSersSuite(t *testing.T) {
	var err error
	login := "test@tmail.io"
	assert := assert.New(t)
	assert.NoError(scope.Init())
	scope.Log, err = logger.New(ioutil.Discard, false)
	assert.NoError(err)

	// drop table users
	assert.NoError(scope.DB.DropTableIfExists(&core.User{}).Error)
	assert.NoError(scope.DB.AutoMigrate(&core.User{}).Error)

	// Get all users should return empty json array
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "http://localhost/foobar", nil)
	r.SetBasicAuth("admin", "admin")

	usersGetAll(w, r)

	b, _ := ioutil.ReadAll(w.Body)
	assert.Equal(200, w.Code, string(b))
	assert.Equal("[]", string(b))

	// Add user
	w = httptest.NewRecorder()
	r, _ = http.NewRequest("POST", "http://localhost/foobar", bytes.NewBufferString(`{"passwd": "passwd", "authRelay": true, "haveMailbox": true, "mailboxQuota": "1G"}`))
	r.SetBasicAuth("admin", "admin")
	ps := httprouter.Params{
		httprouter.Param{"user", login},
	}
	httpcontext.Set(r, "params", ps)

	usersAdd(w, r)
	b, _ = ioutil.ReadAll(w.Body)
	assert.Equal(201, w.Code, string(b))

	// Get users; should return one user
	w = httptest.NewRecorder()
	r, _ = http.NewRequest("GET", "http://localhost/foobar", nil)
	r.SetBasicAuth("admin", "admin")
	usersGetAll(w, r)
	b, _ = ioutil.ReadAll(w.Body)
	assert.Equal(200, w.Code, string(b))
	assert.NotEqual("[]", string(b))
	u := core.User{}

	assert.NoError(json.NewDecoder(bytes.NewReader(b[1 : len(b)-1])).Decode(&u))
	assert.Equal(login, u.Login)
	assert.Equal(true, u.AuthRelay)
	assert.Equal(true, u.HaveMailbox)
	assert.Equal("1G", u.MailboxQuota)

	// Get One users; should return one user
	w = httptest.NewRecorder()
	r, _ = http.NewRequest("GET", "http://localhost/foobar", nil)
	r.SetBasicAuth("admin", "admin")
	ps = httprouter.Params{
		httprouter.Param{"user", login},
	}
	httpcontext.Set(r, "params", ps)
	usersGetOne(w, r)
	b, _ = ioutil.ReadAll(w.Body)
	assert.Equal(200, w.Code, string(b))
	assert.NotEqual("[]", string(b))
	u = core.User{}
	assert.NoError(json.NewDecoder(bytes.NewReader(b)).Decode(&u))
	assert.Equal(login, u.Login)
	assert.Equal(true, u.AuthRelay)
	assert.Equal(true, u.HaveMailbox)
	assert.Equal("1G", u.MailboxQuota)

	// Del user
	w = httptest.NewRecorder()
	r, _ = http.NewRequest("DELETE", "http://localhost/foobar", nil)
	r.SetBasicAuth("admin", "admin")
	ps = httprouter.Params{
		httprouter.Param{"user", login},
	}
	httpcontext.Set(r, "params", ps)
	usersDel(w, r)
	assert.Equal(200, w.Code)
}
