package rest

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/toorop/tmail/api"
	"net/http"
)

// UsersAdd adds an user
func usersAdd(w http.ResponseWriter, r *http.Request) {
	if !authorized(w, r) {
		return
	}
	// parse body
	p := struct {
		Passwd       string `json: "passwd"`
		AuthRelay    bool
		HaveMailbox  bool
		MailboxQuota string
	}{}
	body, err := httpGetBody(r)
	if err != nil {
		httpErrorJson(w, 500, "unable to get JSON body", err.Error())
		return
	}
	if err = json.Unmarshal(body, &p); err != nil {
		httpErrorJson(w, 422, "unable to decode JSON body", err.Error())
		return
	}
	if err = api.UserAdd(mux.Vars(r)["user"], p.Passwd, p.MailboxQuota, p.HaveMailbox, p.AuthRelay); err != nil {
		httpErrorJson(w, 422, "unable to create new user", err.Error())
		return
	}
	w.Header().Set("Location", httpGetScheme()+"://"+r.Host+"/users/"+mux.Vars(r)["user"])
	w.WriteHeader(201)
	return
}

// usersGetAll return all users
func usersGetAll(w http.ResponseWriter, r *http.Request) {
	if !authorized(w, r) {
		return
	}
	users, err := api.UserGetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	js, err := json.Marshal(users)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// usersGetOne return one user
func usersGetOne(w http.ResponseWriter, r *http.Request) {
	if !authorized(w, r) {
		return
	}
	user, err := api.UserGetByLogin(mux.Vars(r)["user"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	js, err := json.Marshal(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)

}

// addUsersHandlers add Users handler to router
func addUsersHandlers(router *mux.Router) {
	// add user
	router.HandleFunc("/users/{user}", usersAdd).Methods("POST")

	// GET /users returns all users
	router.HandleFunc("/users", usersGetAll).Methods("GET")

	// one
	router.HandleFunc("/users/{user}", usersGetOne).Methods("GET")
}
