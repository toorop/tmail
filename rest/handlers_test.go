package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/nbio/httpcontext"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/toorop/tmail/core"
	"github.com/toorop/tmail/logger"
	"github.com/toorop/tmail/scope"
)

func TestHandlerUSers(t *testing.T) {
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

func TestHandlerQueue(t *testing.T) {
	var err error
	assert := assert.New(t)
	assert.NoError(scope.Init())
	scope.Log, err = logger.New(ioutil.Discard, false)
	assert.NoError(err)

	// drop table queue
	assert.NoError(scope.DB.DropTableIfExists(&core.QMessage{}).Error)
	assert.NoError(scope.DB.AutoMigrate(&core.QMessage{}).Error)

	// Get all message in queue should return empty json array
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "http://localhost/foobar", nil)
	r.SetBasicAuth("admin", "admin")
	queueGetMessages(w, r)
	b, _ := ioutil.ReadAll(w.Body)
	assert.Equal(200, w.Code, string(b))
	assert.Equal("[]", string(b))

	// Add message
	message := core.QMessage{
		Uuid:                "uuid",
		Key:                 "key",
		AddedAt:             time.Now(),
		Status:              2,
		DeliveryFailedCount: 0,
	}
	assert.NoError(scope.DB.Create(&message).Error)

	// Get all message
	w = httptest.NewRecorder()
	r, _ = http.NewRequest("GET", "http://localhost/foobar", nil)
	r.SetBasicAuth("admin", "admin")
	queueGetMessages(w, r)
	b, _ = ioutil.ReadAll(w.Body)
	assert.Equal(200, w.Code, string(b))
	assert.NotEqual("[]", string(b))
	m := core.QMessage{}
	assert.NoError(json.NewDecoder(bytes.NewReader(b[1 : len(b)-1])).Decode(&m))
	assert.Equal(m.Uuid, "uuid")

	// Get one
	w = httptest.NewRecorder()
	r, _ = http.NewRequest("GET", "http://localhost/foobar", nil)
	r.SetBasicAuth("admin", "admin")
	ps := httprouter.Params{
		httprouter.Param{"id", fmt.Sprintf("%d", m.Id)},
	}
	httpcontext.Set(r, "params", ps)
	queueGetMessage(w, r)
	b, _ = ioutil.ReadAll(w.Body)
	assert.Equal(200, w.Code, string(b))
	assert.NotEqual("[]", string(b))
	m = core.QMessage{}
	assert.NoError(json.NewDecoder(bytes.NewReader(b)).Decode(&m))
	assert.Equal(m.Uuid, "uuid")

	// discard
	w = httptest.NewRecorder()
	r, _ = http.NewRequest("GET", "http://localhost/foobar", nil)
	r.SetBasicAuth("admin", "admin")
	ps = httprouter.Params{
		httprouter.Param{"id", fmt.Sprintf("%d", m.Id)},
	}
	httpcontext.Set(r, "params", ps)
	queueDiscardMessage(w, r)
	b, _ = ioutil.ReadAll(w.Body)
	assert.Equal(200, w.Code, string(b))

	// bounce
	w = httptest.NewRecorder()
	r, _ = http.NewRequest("GET", "http://localhost/foobar", nil)
	r.SetBasicAuth("admin", "admin")
	ps = httprouter.Params{
		httprouter.Param{"id", fmt.Sprintf("%d", m.Id)},
	}
	httpcontext.Set(r, "params", ps)
	queueBounceMessage(w, r)
	b, _ = ioutil.ReadAll(w.Body)
	assert.Equal(200, w.Code, string(b))

}
