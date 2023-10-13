package handler

import (
	"net/http"

	// added auth import
	"github.com/RICE-COMP318-FALL23/owldb-p1group37/authorization"
	"github.com/RICE-COMP318-FALL23/owldb-p1group37/database"
	"github.com/RICE-COMP318-FALL23/owldb-p1group37/jsonschema"
)

func New(s jsonschema.SchemaValidator) http.Handler {
	auth := authorization.NewAuth()
	ds := database.NewDatabaseService(auth, s)

	mux := http.NewServeMux()
	mux.HandleFunc("/auth", auth.HandleAuthFunctions)
	//slog.Info("auth functions handled")
	mux.HandleFunc("/", ds.DBMethods)

	return mux
}
