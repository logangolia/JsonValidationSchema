package database

import (
	"cmp"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/RICE-COMP318-FALL23/owldb-p1group37/skiplist"
)

// The DatabaseService struct represents the root of the database.
// All documents and collections are stored recursively within the DatabaseService.
// It contains a method to address each of the HTTP methods.
type DatabaseService struct {
	mu          sync.Mutex
	collections skiplist.SkipList[string, Collection]
}

// Placeholder dummy function for skipList implementation
func SetOrUpdate[K cmp.Ordered, V any](key K, currValue V, exists bool) (newValue V, err error) {
	return currValue, nil
}

// NewDatabaseService creates and returns a new DatabaseService struct.
func NewDatabaseService() *DatabaseService {
	minKey := "\x00" // Represents the minimum string key
	maxKey := "\x7F" // Represents the maximum string key
	return &DatabaseService{
		collections: skiplist.NewSkipList[string, Collection](minKey, maxKey),
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

	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func (ds *DatabaseService) HandlePut(w http.ResponseWriter, r *http.Request) {
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
	if len(pathParts)%2 == 0 { // Collection
		collectionName := pathParts[len(pathParts)-1]
		newCollection := NewCollection(collectionName)
		currentItem.(*Document).Collections.Upsert(collectionName, SetOrUpdate)
		response, _ := newCollection.Marshal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(response)
	} else { // Odd length, so it's a document
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
	w.WriteHeader(http.StatusOK)
}
