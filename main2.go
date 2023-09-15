package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Database represents a structure to store documents with metadata.
type Database struct {
	Name      string              `json:"name"`
	Documents map[string]document `json:"documents"`
}

// DatabaseList maintains a thread-safe map of databases.
type DatabaseList struct {
	mu   sync.Mutex
	list map[string]Database
}

// Document represents the structure of stored data along with its metadata.
type document struct {
	mu             sync.Mutex
	Name           string
	Content        string
	CreatedBy      string
	CreatedAt      time.Time
	LastModifiedBy string
	LastModifiedAt time.Time
}

// New initializes a new HTTP handler configured to manage databases.
func New() http.Handler {
	var databases DatabaseList = DatabaseList{list: make(map[string]Database)}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/", databases.HandleDatabases)
	return mux
}

// HandleDatabases is a handler function that routes requests based on their method.
func (dl *DatabaseList) HandleDatabases(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		dl.PutDatabase(w, r)
	case http.MethodGet:
		dl.GetDatabase(w, r)
	}
}

// PutDatabase handles the creation of a new database.
func (dl *DatabaseList) PutDatabase(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	path := strings.Split(r.URL.Path, "/")
	if len(path) != 3 || path[2] == "" {
		slog.Error("PutDatabase: invalid path", "path", path)
		http.Error(w, `"Invalid path"`, http.StatusBadRequest)
		return
	}

	var dbName = path[2]
	var db Database
	db.Name = dbName

	dl.mu.Lock()
	defer dl.mu.Unlock()

	_, exists := dl.list[dbName]
	if exists {
		slog.Error("PutDatabase: database name already exists", "dbName", dbName)
		http.Error(w, `"Database name already exists"`, http.StatusConflict)
		return
	}
	dl.list[db.Name] = db

	slog.Info("PutDatabase: success")
	w.WriteHeader(http.StatusCreated)
}

// GetDatabase retrieves the information of an existing database.
func (dl *DatabaseList) GetDatabase(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	path := strings.Split(r.URL.Path, "/")
	if len(path) != 3 || path[2] == "" {
		slog.Error("GetDatabase: invalid path", "path", path)
		http.Error(w, `"Invalid path"`, http.StatusBadRequest)
		return
	}

	var dbName = path[2]

	dl.mu.Lock()
	defer dl.mu.Unlock()

	db, exists := dl.list[dbName]
	if !exists {
		slog.Error("GetDatabase: database does not exist", "dbName", dbName)
		http.Error(w, `"Database name does not exist"`, http.StatusNotFound)
		return
	}

	jsonDb, err := json.Marshal(db)
	if err != nil {
		slog.Error("GetDatabase: error marshaling database", "database", db)
		http.Error(w, `"Internal server error"`, http.StatusInternalServerError)
		return
	}
	w.Write(jsonDb)

	slog.Info("GetDatabase: found", "dbName", dbName)
}

func main() {
	documents := New()
	var srv http.Server
	srv.Addr = "localhost:3318"
	srv.Handler = documents
	srv.ListenAndServe()

	// Logging and error handling for server initialization can be added here.
}
