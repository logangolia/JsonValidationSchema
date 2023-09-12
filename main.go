// This file is a skeleton for your project. You should replace this
// comment with true documentation.

package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

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

	// // Code to test splitpath and processpath functions
	// ds := setupMockData()

	// tests := []string{
	// 	"/comp318/",
	// 	"/comp318/group1",
	// 	"/comp318/group1/members/rixner",
	// 	"/wrongdb/",
	// 	"/comp318/wrongdoc",
	// 	"/comp318/group1/wrongcollection",
	// }

	// for _, test := range tests {
	// 	db, doc, col, err := ds.processPath(test)
	// 	if err != nil {
	// 		fmt.Printf("Test for path %s failed with error: %s\n", test, err)
	// 	} else {
	// 		fmt.Printf("For path %s, found:\nDatabase: %+v\nDocument: %+v\nCollection: %+v\n\n", test, db, doc, col)
	// 	}
	// }

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

type DatabaseService struct {
	Databases map[string]*Collection
}

type Document struct {
	Data        []byte
	Collections map[string]*Collection
	Metadata    Metadata
}

type Collection struct {
	Documents map[string]*Document
}

type Metadata struct {
	CreatedBy      string
	CreatedAt      time.Time
	LastModifiedBy string
	LastModifiedAt time.Time
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

func (ds *DatabaseService) processPath(path string) (*Collection, *Document, *Collection, error) {
	// Get the individual parts of the path
	parts, err := splitPath(path)
	if err != nil {
		fmt.Println("Error processing path:", err)
		return nil, nil, nil, err
	}

	// If the path is empty, it is invalid
	if len(parts) == 0 {
		return nil, nil, nil, fmt.Errorf("Invalid path")
	}

	// The database is the first part of the path
	database, ok := ds.Databases[parts[0]]
	if !ok {
		return nil, nil, nil, fmt.Errorf("Database not found")
	}

	// Initalize document and collection for return
	var document *Document
	var collection *Collection

	// This will keep track of the current collection context
	currentCollection := database

	// Loop through the rest of the parts of the path
	for i := 1; i < len(parts); i++ {
		if i%2 == 1 {
			// Odd indices are documents
			document, ok = currentCollection.Documents[parts[i]]
			if !ok {
				return nil, nil, nil, fmt.Errorf("Document %s not found", parts[i])
			}
		} else {
			// Even indices are collections
			collection, ok = document.Collections[parts[i]]
			if !ok {
				return nil, nil, nil, fmt.Errorf("Collection %s not found", parts[i])
			}
			// Update current collection context
			currentCollection = collection
		}
	}
	return database, document, collection, nil
}

// Function to create some data for simple testing
// Just for testing processPath and splitPath, we should delete when we can PUT data ourselves.
func setupMockData() *DatabaseService {
	ds := &DatabaseService{
		Databases: map[string]*Collection{
			"comp318": {
				Documents: map[string]*Document{
					"group1": {
						Collections: map[string]*Collection{
							"members": {
								Documents: map[string]*Document{
									"rixner": {},
								},
							},
						},
					},
				},
			},
		},
	}
	return ds
}
