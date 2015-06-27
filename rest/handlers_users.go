package rest

import (
	"encoding/json"
	"net/http"

	"github.com/jinzhu/gorm"
	"github.com/julienschmidt/httprouter"
	"github.com/nbio/httpcontext"
	"github.com/toorop/tmail/api"
)

// usersAdd adds an user
func usersAdd(w http.ResponseWriter, r *http.Request) {
	if !authorized(w, r) {
		return
	}
	p := struct {
		Passwd       string `json: "passwd"`
		AuthRelay    bool   `json: "authRelay"`
		HaveMailbox  bool   `json: "haveMailbox"`
		IsCathall    bool   `json: "isCatchall"`
		MailboxQuota string `json: "mailboxQuota"`
	}{}

	// nil body
	if r.Body == nil {
		httpWriteErrorJson(w, 422, "empty body", "")
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		httpWriteErrorJson(w, 500, "unable to get JSON body", err.Error())
		return
	}

	if err := api.UserAdd(httpcontext.Get(r, "params").(httprouter.Params).ByName("user"), p.Passwd, p.MailboxQuota, p.HaveMailbox, p.AuthRelay, p.IsCathall); err != nil {
		httpWriteErrorJson(w, 422, "unable to create new user", err.Error())
		return
	}
	logInfo(r, "user added "+httpcontext.Get(r, "params").(httprouter.Params).ByName("user"))
	w.Header().Set("Location", httpGetScheme()+"://"+r.Host+"/users/"+httpcontext.Get(r, "params").(httprouter.Params).ByName("user"))
	w.WriteHeader(201)
	return
}

// usersDel delete an user
func usersDel(w http.ResponseWriter, r *http.Request) {
	if !authorized(w, r) {
		return
	}
	err := api.UserDel(httpcontext.Get(r, "params").(httprouter.Params).ByName("user"))
	if err == gorm.RecordNotFound {
		httpWriteErrorJson(w, 404, "no such user "+httpcontext.Get(r, "params").(httprouter.Params).ByName("user"), err.Error())
		return
	}
	if err != nil {
		httpWriteErrorJson(w, 500, "unable to del user "+httpcontext.Get(r, "params").(httprouter.Params).ByName("user"), err.Error())
		return
	}

}

// usersGetAll return all users
func usersGetAll(w http.ResponseWriter, r *http.Request) {
	if !authorized(w, r) {
		return
	}
	users, err := api.UserGetAll()
	if err != nil {
		httpWriteErrorJson(w, 500, "unable to get users", err.Error())
		return
	}
	js, err := json.Marshal(users)
	if err != nil {
		httpWriteErrorJson(w, 500, "JSON encondig failed", err.Error())
		return
	}
	httpWriteJson(w, js)
}

// usersGetOne return one user
func usersGetOne(w http.ResponseWriter, r *http.Request) {
	if !authorized(w, r) {
		return
	}
	user, err := api.UserGetByLogin(httpcontext.Get(r, "params").(httprouter.Params).ByName("user"))
	if err == gorm.RecordNotFound {
		httpWriteErrorJson(w, 404, "no such user "+httpcontext.Get(r, "params").(httprouter.Params).ByName("user"), "")
		return
	}
	if err != nil {
		httpWriteErrorJson(w, 500, "unable to get user "+httpcontext.Get(r, "params").(httprouter.Params).ByName("user"), err.Error())
		return
	}
	js, err := json.Marshal(user)
	if err != nil {
		httpWriteErrorJson(w, 500, "unable to get user "+httpcontext.Get(r, "params").(httprouter.Params).ByName("user"), err.Error())
		return
	}
	httpWriteJson(w, js)
}

// addUsersHandlers add Users handler to router
func addUsersHandlers(router *httprouter.Router) {
	// add user
	router.POST("/users/:user", wrapHandler(usersAdd))

	// get all users
	router.GET("/users", wrapHandler(usersGetAll))

	// get one user
	router.GET("/users/:user", wrapHandler(usersGetOne))

	// del an user
	router.DELETE("/users/:user", wrapHandler(usersDel))
}
