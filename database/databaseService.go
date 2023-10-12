package database

import (
	"cmp"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/RICE-COMP318-FALL23/owldb-p1group37/authorization"
	"github.com/RICE-COMP318-FALL23/owldb-p1group37/skiplist"
)

// The DatabaseService struct represents the root of the database.
// All documents and collections are stored recursively within the DatabaseService.
// It contains a method to address each of the HTTP methods.
type DatabaseService struct {
	mu          sync.Mutex
	auth        *authorization.AuthHandler
	collections skiplist.SkipList[string, Collection]
}

// Placeholder dummy function for skipList implementation
func SetOrUpdate[K cmp.Ordered, V any](key K, currValue V, exists bool) (newValue V, err error) {
	return currValue, nil
}

// NewDatabaseService creates and returns a new DatabaseService struct.
func NewDatabaseService() *DatabaseService {
	var ds DatabaseService
	ds.collections = skiplist.NewSkipList[string, Collection]()
	return &ds
}

func (ds *DatabaseService) dbMethods(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		ds.HandleOptions(w, r)
		return
	}

	if ds.auth.CheckToken(r.Header.Get("Authorization")) != true {
		w.Header().Add("WWW-Authenticate", "Bearer")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	slog.Info("checking token succeeded")

	switch r.Method {
	case http.MethodGet:
		slog.Info("GET called on database")
		ds.HandleGet(w, r)
		slog.Info("GET successful")
	case http.MethodPut:
		slog.Info("PUT called on db")
		ds.HandlePut(w, r)
		slog.Info("PUT successful")
	case http.MethodPost:
		slog.Info("POST called on db")
		ds.HandlePost(w, r)
		slog.Info("POST successful")
	case http.MethodPatch:
		slog.Info("PATCH called on db")
		ds.HandlePatch(w, r)
		slog.Info("PATCH successful")
	case http.MethodDelete:
		slog.Info("DELETE called on db")
		ds.HandleDelete(w, r)
		slog.Info("DELETE successful")
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (ds *DatabaseService) HandleGet(w http.ResponseWriter, r *http.Request) {
	pathParts, err := splitPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Initalize currentItem to the highest-level collection in the path
	var currentItem PathItem
	collection, exists := ds.collections.Find(pathParts[0])
	if !exists {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	currentItem = &collection

	if !exists {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	// Start from index 1 since we've already processed the first collection
	for _, part := range pathParts[1:] {
		nextItem, exists := currentItem.GetChildByName(part)
		if !exists {
			http.Error(w, "Item not found", http.StatusNotFound)
			return
		}
		currentItem = nextItem
	}

	response, err := currentItem.Marshal()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.URL.Query().Get("mode") == "subscribe" {
		subInst := NewSubHandler()
		http.Handle(r.URL.Path, subInst)
		http.HandleFunc(r.URL.Path, subInst.MessageHandler)

	}

	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func (ds *DatabaseService) HandlePut(w http.ResponseWriter, r *http.Request) {
	pathParts, err := splitPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	slog.Info(pathParts[0])

	ds.mu.Lock()
	defer ds.mu.Unlock()

	if len(pathParts) == 1 {
		slog.Info("PUT case database")
		dbName := pathParts[0]
		newCollection := NewCollection(dbName)
		ds.collections.Upsert(dbName, SetOrUpdate)
		response, err := newCollection.Marshal()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		slog.Info("Database Created")
		w.WriteHeader(http.StatusCreated)
		w.Write(response)
		return
	}

	slog.Info("PUT case Collection")

	var currentItem PathItem
	collection, exists := ds.collections.Find(pathParts[0])
	if !exists {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	currentItem = &collection

	if !exists {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	// Traverse through the path until the penultimate item
	for i := 1; i < len(pathParts)-1; i++ {
		nextItem, exists := currentItem.GetChildByName(pathParts[i])
		if !exists {
			http.Error(w, "Path item not found", http.StatusNotFound)
			return
		}
		currentItem = nextItem
	}

	// Handle the final item in the path
	if len(pathParts)%2 == 0 { // Collection
		slog.Info("PUT case Collection")
		collectionName := pathParts[len(pathParts)-1]
		newCollection := NewCollection(collectionName)
		currentItem.(*Document).Collections.Upsert(collectionName, SetOrUpdate)
		response, _ := newCollection.Marshal()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusCreated)
		w.Write(response)
	} else { // Odd length, so it's a document
		slog.Info("PUT case Document")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		docName := pathParts[len(pathParts)-1]
		doc := NewDocument("/"+docName, body, "server", time.Now(), "/v1/"+pathParts[len(pathParts)-2])
		currentItem.(*Collection).Documents.Upsert(docName, SetOrUpdate)
		response, _ := doc.Marshal()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusCreated)
		w.Write(response)
	}
}

func (ds *DatabaseService) HandlePost(w http.ResponseWriter, r *http.Request) {
	pathParts, err := splitPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	var currentItem PathItem
	collection, exists := ds.collections.Find(pathParts[0])
	if !exists {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	currentItem = &collection

	if !exists {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	// Traverse through the path until the penultimate item
	for i := 1; i < len(pathParts)-1; i++ {
		nextItem, exists := currentItem.GetChildByName(pathParts[i])
		if !exists {
			http.Error(w, "Path item not found", http.StatusNotFound)
			return
		}
		currentItem = nextItem
	}

	if len(pathParts)%2 == 0 { // Even length, so it's a collection
		collectionName := pathParts[len(pathParts)-1]
		if _, exists := currentItem.(*Document).Collections.Find(collectionName); exists {
			http.Error(w, "Collection already exists", http.StatusConflict)
			return
		}
		currentItem.(*Document).Collections.Upsert(collectionName, SetOrUpdate)
	} else { // Odd length, so it's a document
		docName := pathParts[len(pathParts)-1]
		_, exists := currentItem.(*Collection).Documents.Find(docName)
		if exists {
			http.Error(w, "Document already exists", http.StatusConflict)
			return
		}

		currentItem.(*Collection).Documents.Upsert(docName, SetOrUpdate)
	}
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusCreated)
}

// Patch needs work
func (ds *DatabaseService) HandlePatch(w http.ResponseWriter, r *http.Request) {
	pathParts, err := splitPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	var currentItem PathItem
	collection, exists := ds.collections.Find(pathParts[0])
	if !exists {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	currentItem = &collection

	if !exists {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	// Traverse through the path until the penultimate item
	for i := 1; i < len(pathParts)-1; i++ {
		nextItem, exists := currentItem.GetChildByName(pathParts[i])
		if !exists {
			http.Error(w, "Path item not found", http.StatusNotFound)
			return
		}
		currentItem = nextItem
	}

	// Handle the final item in the path
	if len(pathParts)%2 == 0 { // Even length, so it's a collection
		collectionName := pathParts[len(pathParts)-1]
		target, exists := currentItem.(*Document).Collections.Find(collectionName)
		if !exists {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}
		decoder := json.NewDecoder(r.Body)
		var updatedCollection Collection
		if err := decoder.Decode(&updatedCollection); err != nil {
			http.Error(w, "Failed to decode request body", http.StatusBadRequest)
			return
		}
		target.URI = updatedCollection.URI
	} else { // Odd length, so it's a document
		docName := pathParts[len(pathParts)-1]
		target, exists := currentItem.(*Collection).Documents.Find(docName)
		if !exists {
			http.Error(w, "Document not found", http.StatusNotFound)
			return
		}
		decoder := json.NewDecoder(r.Body)
		var updatedDoc Document
		if err := decoder.Decode(&updatedDoc); err != nil {
			http.Error(w, "Failed to decode request body", http.StatusBadRequest)
			return
		}
		target.Data = updatedDoc.Data
		target.URI = updatedDoc.URI
	}
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Item updated successfully"))
}

func (ds *DatabaseService) HandleDelete(w http.ResponseWriter, r *http.Request) {
	pathParts, err := splitPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	var currentItem PathItem
	collection, exists := ds.collections.Find(pathParts[0])
	if !exists {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	currentItem = &collection

	if !exists {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	// Traverse through the path until the penultimate item
	for i := 1; i < len(pathParts)-1; i++ {
		nextItem, exists := currentItem.GetChildByName(pathParts[i])
		if !exists {
			http.Error(w, "Path item not found", http.StatusNotFound)
			return
		}
		currentItem = nextItem
	}

	// Handle the final item in the path
	if len(pathParts)%2 == 0 { // Even length, so it's a collection
		collectionName := pathParts[len(pathParts)-1]
		_, exists := currentItem.(*Document).Collections.Find(collectionName)
		if !exists {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}
		ds.collections.Remove(collectionName)
	} else { // Odd length, so it's a document
		docName := pathParts[len(pathParts)-1]
		_, exists := currentItem.(*Collection).Documents.Find(docName)
		if !exists {
			http.Error(w, "Document not found", http.StatusNotFound)
			return
		}
		currentItem.(*Collection).Documents.Remove(docName)
	}
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Item deleted successfully"))
}

func (ds *DatabaseService) HandleOptions(w http.ResponseWriter, r *http.Request) {
	pathParts, err := splitPath(r.URL.Path)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	if len(pathParts) == 1 {
		w.Header().Set("Allow", "PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Methods", "PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.WriteHeader(http.StatusOK)
		return
	}

	var currentItem PathItem
	collection, exists := ds.collections.Find(pathParts[0])
	if !exists {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	currentItem = &collection

	if !exists {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	// Traverse through the path until the penultimate item
	for i := 1; i < len(pathParts)-1; i++ {
		nextItem, exists := currentItem.GetChildByName(pathParts[i])
		if !exists {
			http.Error(w, "Path item not found", http.StatusNotFound)
			return
		}
		currentItem = nextItem
	}

	allowedMethods := "OPTIONS"

	// Determine allowed methods based on the final item in the path
	if len(pathParts)%2 == 0 { // Even length, so it's a collection
		collectionName := pathParts[len(pathParts)-1]
		_, exists := currentItem.(*Document).Collections.Find(collectionName)
		if exists {
			// Collection exists, so GET, DELETE, and PATCH are allowed
			allowedMethods += ", GET, DELETE, PUT,  PATCH"
		} else {
			// Collection does not exist, so POST is allowed
			allowedMethods += ", POST, PUT"
		}
	} else { // Odd length, so it's a document
		docName := pathParts[len(pathParts)-1]
		_, exists := currentItem.(*Collection).Documents.Find(docName)
		if exists {
			// Document exists, so GET, DELETE, PUT, and PATCH are allowed
			allowedMethods += ", GET, DELETE, PUT, PATCH"
		} else {
			// Document does not exist, so POST and PUT are allowed
			allowedMethods += ", POST, PUT"
		}
	}
	w.Header().Set("Allow", allowedMethods)
	w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
	w.WriteHeader(http.StatusOK)
}
