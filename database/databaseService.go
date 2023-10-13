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
	collections skiplist.SkipList[string, *Collection]
}

func GenerateUpdateCheck[K cmp.Ordered, V any](valueToAdd V) skiplist.UpdateCheck[K, V] {
	return func(key K, currValue V, exists bool) (newValue V, err error) {
		// In this case, whether the item exists or not, it will set/update the value to valueToAdd.
		return valueToAdd, nil
	}
}

// NewDatabaseService creates and returns a new DatabaseService struct.
func NewDatabaseService(auth *authorization.AuthHandler) *DatabaseService {
	var ds DatabaseService
	ds.collections = skiplist.NewSkipList[string, *Collection]()
	ds.auth = auth
	return &ds
}

func (ds *DatabaseService) DBMethods(w http.ResponseWriter, r *http.Request) {
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
	// Set header for response.
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Parse the path.
	pathParts, err := splitPath(r.URL.Path)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Lock the database.
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Initalize currentItem to the database in the path.
	var currentItem PathItem
	collection, exists := ds.collections.Find(pathParts[1])
	if !exists {
		sendErrorResponse(w, http.StatusNotFound, "\"Database does not exist\"")
		return
	}
	currentItem = collection

	// Start from index 2 since we've already processed the database.
	for _, part := range pathParts[2:] {
		nextItem, exists := currentItem.GetChildByName(part)
		if !exists {
			if len(pathParts)%2 == 0 {
				sendErrorResponse(w, http.StatusNotFound, "\"Collection does not exist\"")
				return
			} else {
				sendErrorResponse(w, http.StatusNotFound, "\"Document does not exist\"")
				return
			}
		}
		currentItem = nextItem
	}

	// Marshall the item.
	response, err := currentItem.Marshal()
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Handle subscribe.
	if r.URL.Query().Get("mode") == "subscribe" {
		subInst := NewSubHandler()
		http.Handle(r.URL.Path, subInst)
		http.HandleFunc(r.URL.Path, subInst.MessageHandler)
	}

	// Successful GET request.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func (ds *DatabaseService) HandlePut(w http.ResponseWriter, r *http.Request) {
	// Set header for response.
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Parse the path.
	pathParts, err := splitPath(r.URL.Path)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info(pathParts[1])

	// Lock the databse.
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Edge case if we are putting a database.
	if len(pathParts) == 2 {
		slog.Info("PUT case database")
		collectionName := pathParts[1]
		// Check if the database already exists
		_, exists := ds.collections.Find(collectionName)
		if exists {
			sendErrorResponse(w, http.StatusBadRequest, "\"unable to create database "+collectionName+": exists\"")
			return
		}
		newCollection := NewCollection(collectionName, r.URL.Path)
		updateFunc := GenerateUpdateCheck[string, *Collection](newCollection)
		ds.collections.Upsert(collectionName, updateFunc)
		response, err := newCollection.MarshalURI()
		if err != nil {
			sendErrorResponse(w, http.StatusNotFound, err.Error())
			return
		}
		slog.Info("Database Created")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(response)
		return
	}

	// Get the top-level database of the path
	var currentItem PathItem
	database, exists := ds.collections.Find(pathParts[1])
	if !exists {
		sendErrorResponse(w, http.StatusNotFound, "\"Collection does not exist\"")
		return
	}
	currentItem = database

	// Traverse through the path until the penultimate item
	for i := 2; i < len(pathParts)-1; i++ {
		nextItem, exists := currentItem.GetChildByName(pathParts[i])
		if !exists {
			if len(pathParts)%2 == 0 {
				sendErrorResponse(w, http.StatusNotFound, "\"Document does not exist\"")
				return
			} else {
				sendErrorResponse(w, http.StatusNotFound, "\"Collection does not exist\"")
				return
			}
		}
		currentItem = nextItem
	}

	// Handle the final item in the path
	if len(pathParts)%2 == 0 { // Collection
		slog.Info("PUT case Collection")
		collectionName := pathParts[len(pathParts)-1]
		newCollection := NewCollection(collectionName, r.URL.Path)
		updateFunc := GenerateUpdateCheck[string, *Collection](newCollection)
		_, upsertErr := currentItem.(*Document).Collections.Upsert(collectionName, updateFunc)
		if upsertErr != nil {
			sendErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		response, err := newCollection.MarshalURI()
		if err != nil {
			sendErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(response)
	} else { // Document
		slog.Info("PUT case Document")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			sendErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer r.Body.Close()
		docName := pathParts[len(pathParts)-1]
		// Check if the document is being created for the first time or being overriden
		override := false
		_, exists := currentItem.(*Collection).Documents.Find(docName)
		if exists {
			override = true
		}
		var data interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			sendErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		newDocument := NewDocument("/"+docName, data, "server", time.Now(), r.URL.Path)
		updateFunc := GenerateUpdateCheck[string, *Document](newDocument)
		_, upsertErr := currentItem.(*Collection).Documents.Upsert(docName, updateFunc)
		if upsertErr != nil {
			sendErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		response, err := newDocument.MarshalURI()
		if err != nil {
			sendErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// Overriding/Creating a document have different response codes
		if override {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusCreated)
		}
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
	collection, exists := ds.collections.Find(pathParts[1])
	if !exists {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	currentItem = collection

	// Traverse through the path until the penultimate item
	for i := 2; i < len(pathParts)-1; i++ {
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
		newCollection := NewCollection(collectionName, r.URL.Path)
		updateFunc := GenerateUpdateCheck[string, *Collection](newCollection)
		currentItem.(*Document).Collections.Upsert(collectionName, updateFunc)
	} else { // Odd length, so it's a document
		docName := pathParts[len(pathParts)-1]
		_, exists := currentItem.(*Collection).Documents.Find(docName)
		if exists {
			http.Error(w, "Document already exists", http.StatusConflict)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var data interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid JSON format"))
			return
		}
		newDocument := NewDocument(docName, data, "server", time.Now(), r.URL.Path)
		updateFunc := GenerateUpdateCheck[string, *Document](newDocument)
		currentItem.(*Collection).Documents.Upsert(docName, updateFunc)
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
	collection, exists := ds.collections.Find(pathParts[1])
	if !exists {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	currentItem = collection

	if !exists {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	// Traverse through the path until the penultimate item
	for i := 2; i < len(pathParts)-1; i++ {
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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// Parse the path.
	pathParts, err := splitPath(r.URL.Path)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Lock the databse.
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Edge case if we are deleting a database.
	if len(pathParts) == 2 {
		slog.Info("Delete case database")
		collectionName := pathParts[1]
		// Check if the database exists
		_, exists := ds.collections.Find(collectionName)
		if !exists {
			sendErrorResponse(w, http.StatusNotFound, "\"Database does not exist\"")
			return
		}
		_, ok := ds.collections.Remove(collectionName)
		if !ok {
			sendErrorResponse(w, http.StatusInternalServerError, "\"Failed to remove database\"")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Get the top-level database of the path
	var currentItem PathItem
	database, exists := ds.collections.Find(pathParts[1])
	if !exists {
		sendErrorResponse(w, http.StatusNotFound, "\"unable to create/replace document: not found\"")
		return
	}
	currentItem = database

	// Traverse through the path until the penultimate item
	for i := 2; i < len(pathParts)-1; i++ {
		nextItem, exists := currentItem.GetChildByName(pathParts[i])
		if !exists {
			if len(pathParts)%2 == 0 {
				sendErrorResponse(w, http.StatusNotFound, "\"Collection does not exist\"")
				return
			} else {
				sendErrorResponse(w, http.StatusNotFound, "\"Document does not exist\"")
				return
			}
		}
		currentItem = nextItem
	}

	// Handle the final item in the path
	if len(pathParts)%2 == 0 { // Collection
		collectionName := pathParts[len(pathParts)-1]
		_, exists := currentItem.(*Document).Collections.Find(collectionName)
		if !exists {
			sendErrorResponse(w, http.StatusNotFound, "\"Collection does not exist\"")
			return
		}
		_, ok := currentItem.(*Document).Collections.Remove(collectionName)
		if !ok {
			sendErrorResponse(w, http.StatusInternalServerError, "\"Failed to remove collection\"")
			return
		}
	} else { // Document
		docName := pathParts[len(pathParts)-1]
		_, exists := currentItem.(*Collection).Documents.Find(docName)
		if !exists {
			sendErrorResponse(w, http.StatusNotFound, "\"Document does not exist\"")
			return
		}
		_, ok := currentItem.(*Collection).Documents.Remove(docName)
		if !ok {
			sendErrorResponse(w, http.StatusInternalServerError, "\"Failed to remove document\"")
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
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
	collection, exists := ds.collections.Find(pathParts[1])
	if !exists {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	currentItem = collection

	if !exists {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	// Traverse through the path until the penultimate item
	for i := 2; i < len(pathParts)-1; i++ {
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

func sendErrorResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(message))
}
