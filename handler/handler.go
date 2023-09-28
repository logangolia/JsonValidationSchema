package handler

import (
	"main/database"
	"net/http"
)

func NewHandler() http.Handler {
	mux := http.NewServeMux()

	dbService := database.NewDatabaseService()

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
