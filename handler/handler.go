package handler

import (
	"net/http"

	// added auth import
	"github.com/RICE-COMP318-FALL23/owldb-p1group37/authorization"
	"github.com/RICE-COMP318-FALL23/owldb-p1group37/database"
)

func NewHandler() http.Handler {
	mux := http.NewServeMux()

	dbService := database.NewDatabaseService()

	// Route /auth URL path to authHandler function
	mux.HandleFunc("/auth", authorization.AuthHandler)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			dbService.HandleGet(w, r)
		case http.MethodPut:
			dbService.HandlePut(w, r)
		case http.MethodPost:
			dbService.HandlePost(w, r)
		case http.MethodPatch:
			dbService.HandlePatch(w, r)
		case http.MethodDelete:
			dbService.HandleDelete(w, r)
		case http.MethodOptions:
			dbService.HandleOptions(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return mux
}
