package rest

import (
	"encoding/json"
	"fmt"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/toorop/tmail/api"
	"github.com/toorop/tmail/scope"
	"log"
	"net/http"
)

func userGetAll(w http.ResponseWriter, r *http.Request) {
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

func userGet(w http.ResponseWriter, r *http.Request) {
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

func LaunchServer() {
	router := mux.NewRouter()
	//router.HandleFunc("/", HomeHandler)
	router.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "Welcome to the home page!")
	})

	// User
	// all
	router.HandleFunc("/users", userGetAll).Methods("GET")
	// one
	router.HandleFunc("/user/{user}", userGet).Methods("GET")

	n := negroni.New(negroni.NewRecovery(), NewLogger())
	n.UseHandler(router)
	//n.Run(fmt.Sprintf("%s:%d", scope.Cfg.GetRestServerIp(), scope.Cfg.GetRestServerPort()))
	addr := fmt.Sprintf("%s:%d", scope.Cfg.GetRestServerIp(), scope.Cfg.GetRestServerPort())
	scope.Log.Info("http (REST server) " + addr + " launched")
	log.Fatalln(http.ListenAndServe(addr, n))
}
