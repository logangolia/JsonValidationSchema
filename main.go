// This file is a skeleton for your project. You should replace this
// comment with true documentation.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type DatabaseService struct {
	mu        sync.Mutex
	databases map[string]*Collection
}

type Document struct {
	name        string
	data        []byte
	collections map[string]*Collection
	metadata    Metadata
}

type Collection struct {
	name      string
	documents map[string]*Document
}

type Metadata struct {
	createdBy      string
	createdAt      time.Time
	lastModifiedBy string
	lastModifiedAt time.Time
}

// "github.com/santhosh-tekuri/jsonschema/v5/httploader"

func main() {
	var server http.Server
	var port int
	var err error

	// Your code goes here.

	// Set port as defined at -p, with default port 3318
	portPtr := flag.Int("p", 3318, "port on which the server will listen")
	flag.Parse()

	port = *portPtr

	// Set server address based on port
	server.Addr = ":" + fmt.Sprintf("%d", port)

	// Assign the handler to the server
	server.Handler = NewHandler()

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

func NewHandler() http.Handler {
	// Create server mux
	mux := http.NewServeMux()

	// Create new highest-level database, a DatabaseService
	dbService := &DatabaseService{
		databases: make(map[string]*Collection),
	}

	// One HandleFunc with switch cases for Get/Put
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			dbService.HandleGet(w, r)
		case http.MethodPut:
			dbService.HandlePut(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return mux
}

func (ds *DatabaseService) HandleGet(w http.ResponseWriter, r *http.Request) {
	// Get the parts of the path from splitPath
	pathParts, err := splitPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If the path is empty it is invalid
	if len(pathParts) == 0 {
		http.Error(w, "Empty Path", http.StatusBadRequest)
		return
	}

	// Lock for the rest of the func
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Check if the database exists
	database, dbExists := ds.databases[pathParts[0]]
	if !dbExists {
		http.Error(w, "Database does not exist", http.StatusBadRequest)
	}

	if len(pathParts) == 1 {
		// We are getting a database

		// Loop through all the documents in the database to marshall and write
		var names []string
		for name := range database.documents {
			names = append(names, name)
		}
		responseData, ok := json.Marshal(names)
		if ok != nil {
			http.Error(w, "Error marshaling", http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(responseData)
		}
	} else {
		// We are getting a document

		// Check if the document exists
		document, documentExists := database.documents[pathParts[1]]
		if documentExists {
			// Marshall and write the document
			responseData, ok := json.Marshal(document)
			if ok != nil {
				http.Error(w, "Error marshaling", http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write(responseData)
			}
		} else {
			http.Error(w, "Document does not exist", http.StatusBadRequest)
		}
	}
	return
}

func (ds *DatabaseService) HandlePut(w http.ResponseWriter, r *http.Request) {
	// Get the parts of the path from splitPath
	pathParts, err := splitPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Invalid if path is empty
	if len(pathParts) == 0 {
		http.Error(w, "Empty Path", http.StatusBadRequest)
		return
	}

	// Lock for the rest of the func
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if len(pathParts) == 1 {
		// We're dealing with a database
		databaseName := pathParts[0]

		// Check if the databaseName already exists in ds.databases
		_, databaseExists := ds.databases[databaseName]

		// If the database doesn't exist, create a new one
		if databaseExists {
			http.Error(w, "Database already exists", http.StatusConflict)
		} else {
			ds.databases[databaseName] = &Collection{
				documents: make(map[string]*Document),
			}
			w.WriteHeader(http.StatusCreated) // Indicate that a new database was created
		}
	} else {
		// We're dealing with a document
		// Check that the database where the document will go exists
		database, ok := ds.databases[pathParts[0]]
		if !ok {
			http.Error(w, "Invalid Database", http.StatusNotFound)
			return
		}

		// Read the body of the message
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		// We're dealing with a document
		database.documents[pathParts[1]] = &Document{
			data:        body,
			collections: make(map[string]*Collection),
			metadata: Metadata{
				createdBy:      "server",
				createdAt:      time.Now(),
				lastModifiedBy: "server",
				lastModifiedAt: time.Now(),
			},
		}
		w.WriteHeader(http.StatusCreated) // Indicate that a new document was created
	}
	return
}

// splitPath splits the given path into its components.
func splitPath(path string) ([]string, error) {
	// Remove leading and trailing slashes if they exist.
	trimmedPath := strings.Trim(path, "/")

	// Split the path by slashes.
	parts := strings.Split(trimmedPath, "/")

	// Loop over parts to translate percent-encoded characters.
	for i, part := range parts {
		decodedPart, err := url.QueryUnescape(part)
		if err != nil {
			return nil, fmt.Errorf("Error decoding path part: %s", err)
		}
		parts[i] = decodedPart
	}

	// The returned slice removes the leading and trailing slashes and decodes any percent-encoded values.
	return parts, nil
}
