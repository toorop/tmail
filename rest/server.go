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
	"os"
	"path"
	"path/filepath"
)

func userGetAll(w http.ResponseWriter, r *http.Request) {
	if !isAuthorized(w, r) {
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

	//http.Handle("/users", HttpAuthBasic("toorop", "test")(router))
	router.HandleFunc("/users", userGetAll).Methods("GET")

	// one
	router.HandleFunc("/users/{user}", userGet).Methods("GET")

	// Server
	n := negroni.New(negroni.NewRecovery(), NewLogger())
	n.UseHandler(router)
	//n.Run(fmt.Sprintf("%s:%d", scope.Cfg.GetRestServerIp(), scope.Cfg.GetRestServerPort()))
	addr := fmt.Sprintf("%s:%d", scope.Cfg.GetRestServerIp(), scope.Cfg.GetRestServerPort())

	// TLS
	if scope.Cfg.GetRestServerIsTls() {
		scope.Log.Info("httpd " + addr + " TLS launched")
		log.Fatalln(http.ListenAndServeTLS(addr, path.Join(getBasePath(), "ssl/server.crt"), path.Join(getBasePath(), "ssl/server.key"), n))
	} else {
		scope.Log.Info("httpd " + addr + " launched")
		log.Fatalln(http.ListenAndServe(addr, n))
	}
}

func getBasePath() string {
	p, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	return p
}
