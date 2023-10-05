package database

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/RICE-COMP318-FALL23/owldb-p1group37/skiplist"
)

// The Document struct represents a document in a database.
type Document struct {
	Name        string `json:"path"`
	Data        []byte `json:"doc"`
	Collections skiplist.SkipList[string, Collection]
	Metadata    Metadata `json:"meta"`
	URI         string   `json:"uri"`
}

// NewDocument creates and returns a new Document struct based on the inputs.
func NewDocument(name string, data []byte, user string, time time.Time, uriPrefix string) *Document {
	minKey := "\x00" // Represents the minimum string key
	maxKey := "\x7F" // Represents the maximum string key
	return &Document{
		Name:        name,
		Data:        data,
		Collections: skiplist.NewSkipList[string, Collection](minKey, maxKey),
		Metadata:    *NewMetadata(user, time),
		URI:         uriPrefix + name,
	}
}

// GetChildByName implements the function from the PathItem interface.
// If it exists, it returns the collection and true, otherwise nil and false.
func (d *Document) GetChildByName(name string) (PathItem, bool) {
	child, exists := d.Collections.Find(name)
	if exists {
		return &child, true
	}
	return nil, false
}

// Marshal implements the function from the PathItem interface.
// Calling Marshal() marshals and returns the document as well as an error.
func (d *Document) Marshal() ([]byte, error) {
	response, err := json.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("Error marshaling document: %w", err)
	}
	return response, nil
}
