// This file is a skeleton for your project. You should replace this
// comment with true documentation.

package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// a Database stores documents
type Database struct {
	Name      string              `json:"name"`
	Documents map[string]document `json:"documents"`
}

// DatabaseList
type DatabaseList struct {
	mu   sync.Mutex
	list map[string]Database
}

// documents store json content in raw bytes
// and collections, but tbimplemented
type document struct {
	mu             sync.Mutex
	Name           string
	Content        string
	CreatedBy      string
	CreatedAt      time.Time
	LastModifiedBy string
	LastModifiedAt time.Time
}

// Creates new NoSQL backend DB server
func New() http.Handler {
	// create list of Databases
	var Databases DatabaseList = DatabaseList{list: make(map[string]Database)}

	// Set the handlers for the appropriate paths
	mux := http.NewServeMux()
	// TODO: Specify the paths for Databases
	mux.HandleFunc("/v1/", Databases.HandleDatabases)
	// mux.HandleFunc("", Database.HandleDocument)

	return mux
}

func (dl *DatabaseList) HandleDatabases(w http.ResponseWriter, r *http.Request) {
	// dispatch on method - GET or PUT
	switch r.Method {
	case http.MethodPut:
		// create new DB
		dl.PutDatabase(w, r)
	case http.MethodGet:
		// retrieve existing DB
		dl.GetDatabase(w, r)
	}
}

// Create a new Database
func (dl *DatabaseList) PutDatabase(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// parse resource request
	path := strings.Split(r.URL.Path, "/")
	// check if url is valid
	if len(path) != 3 || path[2] == "" {
		slog.Error("PutDatabase: invalid path", "path", path)
		http.Error(w, `"Invalid path"`, http.StatusBadRequest)
		return
	}
	// TODO: url string conversion
	var dbName = path[2]

	// create Database
	var Database Database
	Database.Name = dbName

	dl.mu.Lock()
	defer dl.mu.Unlock()

	// check if Database exists
	_, exists := dl.list[dbName]
	if exists {
		slog.Error("PutDatabase: db name already exists", "dbName", dbName)
		http.Error(w, `"Database name already exists"`, http.StatusConflict)
		return
	}
	// add the Database to db list
	dl.list[Database.Name] = Database

	slog.Info("PutDatabase: success")
	w.WriteHeader(http.StatusCreated)
}

func (dl *DatabaseList) GetDatabase(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// parse resource request
	path := strings.Split(r.URL.Path, "/")
	// check if url is valid
	if len(path) != 3 || path[2] == "" {
		slog.Error("PutDatabase: invalid path", "path", path)
		http.Error(w, `"Invalid path"`, http.StatusBadRequest)
		return
	}
	// TODO: url string conversion
	var dbName = path[2]

	dl.mu.Lock()
	defer dl.mu.Unlock()

	// check if Database exists
	Database, exists := dl.list[dbName]
	if !exists {
		slog.Error("GetDatabase: db does not exist", "dbName", dbName)
		http.Error(w, `"Database name does not exist"`, http.StatusNotFound)
		return
	}
	// send Database as json
	jsonDb, err := json.Marshal(Database)
	if err != nil {
		// This should never happen
		slog.Error("GetDatabase: error marshaling Database", "Database", Database)
		http.Error(w, `"internal server error"`, http.StatusInternalServerError)
		return
	}
	w.Write(jsonDb)

	slog.Info("GetDatabase: found", "dbName", dbName)
}

func main() {
	var server http.Server
	var port int
	var err error

	// Your code goes here.

	// var "documents" returns a handler for both docs and Databases. name change needed
	documents := New()
	var srv http.Server
	srv.Addr = "localhost:3318"
	srv.Handler = documents
	srv.ListenAndServe()

	// The following code should go last and remain unchanged.
	// Note that you must actually initialize 'server' and 'port'
	// before this.

	// signal.Notify requires the channel to be buffered
	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt, syscall.SIGTERM)
	go func() {
		// Wait for Ctrl-C signal
		<-ctrlc
		server.Close()
	}()

	// Start server
	slog.Info("Listening", "port", port)
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		slog.Error("Server closed", "error", err)
	} else {
		slog.Info("Server closed", "error", err)
	}
}
