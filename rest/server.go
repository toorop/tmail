package rest

import (
	"fmt"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/toorop/tmail/scope"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

const (
	// Max size of the posted body
	body_read_limit = 1048576
)

// LaunchServer launches HTTP server
func LaunchServer() {
	router := mux.NewRouter()
	//router.HandleFunc("/", HomeHandler)
	router.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "coucou")
	})

	// Users handlers
	addUsersHandlers(router)

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

// getBasePath is a helper for retrieving app path
func getBasePath() string {
	p, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	return p
}
