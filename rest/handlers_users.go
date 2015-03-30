package rest

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/toorop/tmail/api"
	"io"
	"io/ioutil"
	"net/http"
)

// UsersAdd adds an user
func usersAdd(w http.ResponseWriter, r *http.Request) {
	if !authorized(w, r) {
		return
	}

	type payload struct {
		Passwd string `json: "passwd"`
		//AuthRelay    bool
		//HaveMailbox  bool
		//MailboxQuota string
	}

	p := &payload{}

	// get
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, body_read_limit))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logDebug(r, string(body))
	if err := json.Unmarshal(body, p); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	logDebug(r, p.Passwd)

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
