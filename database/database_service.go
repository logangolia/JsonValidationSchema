package database

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

type DatabaseService struct {
	mu          sync.Mutex
	collections map[string]*Collection
}

func NewDatabaseService() *DatabaseService {
	return &DatabaseService{
		collections: make(map[string]*Collection),
	}
}

func (ds *DatabaseService) HandleGet(w http.ResponseWriter, r *http.Request) {
	// Get the parts of the path from splitPath
	pathParts, err := splitPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Lock for the rest of the func
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Check if the collection exists
	collection, dbExists := ds.collections[pathParts[0]]
	if !dbExists {
		http.Error(w, "Database does not exist", http.StatusNotFound)
	}

	if len(pathParts) == 1 {
		// We are getting a collection
		response, err := marshalCollection(collection)
		if err != nil {
			http.Error(w, "Failed to marshal collection", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	} else {
		// We are getting a document

		// Check if the document exists
		document, documentExists := collection.Documents[pathParts[1]]
		if documentExists {
			response, err := marshalDocument(document)
			if err != nil {
				http.Error(w, "Failed to marshal collection", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write(response)
		} else {
			http.Error(w, "Document does not exist", http.StatusNotFound)
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

	// Lock for the rest of the func
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if len(pathParts) == 1 {
		// We're dealing with a database
		collectionName := pathParts[0]

		// Check if the databaseName already exists in ds.databases
		_, collectionExists := ds.collections[collectionName]

		// If the database doesn't exist, create a new one
		if collectionExists {
			http.Error(w, "Database already exists", http.StatusBadRequest)
		} else {
			ds.collections[collectionName] = NewCollection(collectionName)
			response, err := marshalCollection(ds.collections[collectionName])
			if err != nil {
				http.Error(w, "Error marshaling", http.StatusInternalServerError)
			} else {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				w.Write(response)
			}
		}
	} else {
		// We're dealing with a document
		// Check that the collection where the document will go exists
		collection, ok := ds.collections[pathParts[0]]
		if !ok {
			http.Error(w, "Invalid Database", http.StatusBadRequest)
			return
		}

		// Read the body of the message
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		// Process document
		collection.Documents[pathParts[1]] = NewDocument("/"+pathParts[1], body, "server", time.Now(), "/v1/"+collection.Name)
		response, err := marshalDocument(collection.Documents[pathParts[1]])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write(response)
		}
	}
	return
}

func (ds *DatabaseService) HandlePost(w http.ResponseWriter, r *http.Request) {
	// Parse the path
	pathParts, err := splitPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	if len(pathParts) == 1 {
		// We're dealing with a collection
		collectionName := pathParts[0]

		// Check if the collectionName already exists
		_, collectionExists := ds.collections[collectionName]

		if collectionExists {
			http.Error(w, "Collection already exists", http.StatusConflict)
			return
		}

		// Create a new collection
		ds.collections[collectionName] = NewCollection(collectionName)
		response, err := marshalCollection(ds.collections[collectionName])
		if err != nil {
			http.Error(w, "Error marshaling collection", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(response)
	} else {
		// We're dealing with a document within a collection
		collection, ok := ds.collections[pathParts[0]]
		if !ok {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}

		// Check if the document already exists
		_, docExists := collection.Documents[pathParts[1]]
		if docExists {
			http.Error(w, "Document already exists", http.StatusConflict)
			return
		}

		// Read the request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		// Create a new document and add it to the collection
		collection.Documents[pathParts[1]] = NewDocument("/"+pathParts[1], body, "server", time.Now(), "/v1/"+collection.Name)

		responseData, err := json.Marshal(collection.Documents[pathParts[1]])
		if err != nil {
			http.Error(w, "Error marshaling document", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(responseData)
	}
}

func (ds *DatabaseService) HandlePatch(w http.ResponseWriter, r *http.Request) {
	// Parse the path
	pathParts, err := splitPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if the path is empty
	if len(pathParts) == 0 {
		http.Error(w, "Empty Path", http.StatusBadRequest)
		return
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	if len(pathParts) == 1 {
		// We're dealing with a collection
		collectionName := pathParts[0]

		// Check if the collection exists
		collection, ok := ds.collections[collectionName]
		if !ok {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}

		// Apply the patch to the collection (assuming you have a method on collection for this)
		// For simplicity, we'll assume a PATCH just updates the collection's URI
		decoder := json.NewDecoder(r.Body)
		var updatedCollection Collection
		err := decoder.Decode(&updatedCollection)
		if err != nil {
			http.Error(w, "Failed to decode request body", http.StatusBadRequest)
			return
		}

		collection.URI = updatedCollection.URI

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Collection updated successfully"))

	} else {
		// We're dealing with a document within a collection
		collection, ok := ds.collections[pathParts[0]]
		if !ok {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}

		// Check if the document exists
		doc, ok := collection.Documents[pathParts[1]]
		if !ok {
			http.Error(w, "Document not found", http.StatusNotFound)
			return
		}

		// Apply the patch to the document
		decoder := json.NewDecoder(r.Body)
		var updatedDoc Document
		err := decoder.Decode(&updatedDoc)
		if err != nil {
			http.Error(w, "Failed to decode request body", http.StatusBadRequest)
			return
		}

		// As an example, let's say we only allow updating the Data of a document using PATCH
		doc.Data = updatedDoc.Data

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Document updated successfully"))
	}
}

func (ds *DatabaseService) HandleDelete(w http.ResponseWriter, r *http.Request) {
	// Parse the path
	pathParts, err := splitPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()
	if len(pathParts) == 1 {
		// We're dealing with a collection
		collectionName := pathParts[0]

		// Check if the collection exists
		_, collectionExists := ds.collections[collectionName]

		if !collectionExists {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}

		// Delete the collection
		delete(ds.collections, collectionName)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Collection deleted successfully"))

	} else {
		// We're dealing with a document within a collection
		collection, ok := ds.collections[pathParts[0]]
		if !ok {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}

		// Check if the document exists
		_, docExists := collection.Documents[pathParts[1]]
		if !docExists {
			http.Error(w, "Document not found", http.StatusNotFound)
			return
		}

		// Delete the document from the collection
		delete(collection.Documents, pathParts[1])
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Document deleted successfully"))
	}
}

func (ds *DatabaseService) HandleOptions(w http.ResponseWriter, r *http.Request) {
	// Parse the path
	pathParts, err := splitPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	allowedMethods := "OPTIONS"

	if len(pathParts) == 1 {
		// We're dealing with a collection
		collectionName := pathParts[0]

		// Check if the collection exists
		_, ok := ds.collections[collectionName]
		if ok {
			// If the collection exists, then we can perform GET, PUT, PATCH, and DELETE on it
			allowedMethods = "OPTIONS, GET, PUT, PATCH, DELETE"
		} else {
			// If the collection doesn't exist, it means we can create it using PUT
			allowedMethods = "OPTIONS, PUT"
		}

	} else if len(pathParts) == 2 {
		// We're dealing with a document within a collection
		_, ok := ds.collections[pathParts[0]]
		if ok {
			// If the collection exists, check if the document exists
			_, docExists := ds.collections[pathParts[0]].Documents[pathParts[1]]
			if docExists {
				// If the document exists, then we can perform GET, PUT, PATCH, and DELETE on it
				allowedMethods = "OPTIONS, GET, PUT, PATCH, DELETE"
			} else {
				// If the document doesn't exist, it means we can create it using PUT
				allowedMethods = "OPTIONS, PUT"
			}
		}
	}

	w.Header().Set("Allow", allowedMethods)
	w.WriteHeader(http.StatusOK)
}
