package rest

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/toorop/tmail/api"
	"net/http"
)

// UsersAdd adds an user
func usersAdd(w http.ResponseWriter, r *http.Request) {
	if !authorized(w, r) {
		return
	}
	p := struct {
		Passwd       string `json: "passwd"`
		AuthRelay    bool   `json: "authRelay"`
		HaveMailbox  bool   `json: "haveMailbox"`
		MailboxQuota string `json: "mailboxQuota"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		httpWriteErrorJson(w, 500, "unable to get JSON body", err.Error())
		return
	}

	if err := api.UserAdd(mux.Vars(r)["user"], p.Passwd, p.MailboxQuota, p.HaveMailbox, p.AuthRelay); err != nil {
		httpWriteErrorJson(w, 422, "unable to create new user", err.Error())
		return
	}
	logInfo(r, "user added "+mux.Vars(r)["user"])
	w.Header().Set("Location", httpGetScheme()+"://"+r.Host+"/users/"+mux.Vars(r)["user"])
	w.WriteHeader(201)
	return
}

// usersDel delete an user
func usersDel(w http.ResponseWriter, r *http.Request) {
	if !authorized(w, r) {
		return
	}
	if err := api.UserDel(mux.Vars(r)["user"]); err != nil {
		httpWriteErrorJson(w, 500, "unable to delete user "+mux.Vars(r)["user"], err.Error())
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
	user, err := api.UserGetByLogin(mux.Vars(r)["user"])
	if err == gorm.RecordNotFound {
		httpWriteErrorJson(w, 404, "no such user "+mux.Vars(r)["user"], "")
		return
	}
	if err != nil {
		httpWriteErrorJson(w, 500, "unable to get user "+mux.Vars(r)["user"], err.Error())
		return
	}
	js, err := json.Marshal(user)
	if err != nil {
		httpWriteErrorJson(w, 500, "unable to get user "+mux.Vars(r)["user"], err.Error())
		return
	}
	httpWriteJson(w, js)
}

// addUsersHandlers add Users handler to router
func addUsersHandlers(router *mux.Router) {
	// add user
	router.HandleFunc("/users/{user}", usersAdd).Methods("POST")

	// get all users
	router.HandleFunc("/users", usersGetAll).Methods("GET")

	// get one user
	router.HandleFunc("/users/{user}", usersGetOne).Methods("GET")

	// del an user
	router.HandleFunc("/users/{user}", usersDel).Methods("DELETE")
}
