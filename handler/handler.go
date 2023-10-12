package handler

import (
	"net/http"

	// added auth import
	"github.com/RICE-COMP318-FALL23/owldb-p1group37/authorization"
	"github.com/RICE-COMP318-FALL23/owldb-p1group37/database"
)

func New() http.Handler {
	auth := authorization.NewAuth()
	ds := database.NewDatabaseService(auth)

	mux := http.NewServeMux()
	mux.HandleFunc("/auth", auth.HandleAuthFunctions)
	mux.HandleFunc("/", ds.DBMethods)

	return mux
}
