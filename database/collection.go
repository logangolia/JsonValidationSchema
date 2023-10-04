package database

import (
	"encoding/json"

	"github.com/RICE-COMP318-FALL23/owldb-p1group37/skiplist"
)

// The Collection struct represents a collection in a database.
type Collection struct {
	Name      string
	Documents *skiplist.SkipList
	URI       string `json:"uri"`
}

// NewCollection creates and returns a new Collection struct with the given name.
func NewCollection(name string) *Collection {
	return &Collection{
		Name:      name,
		Documents: skiplist.NewSkipList(),
		URI:       name,
	}
}

// GetChildByName implements the function from the PathItem interface.
// If it exists, it returns the document and true, otherwise nil and false.
func (c *Collection) GetChildByName(name string) (PathItem, bool) {
	child, exists := c.Documents.Find(name)
	return child, exists
}

// Marshal implements the function from the PathItem interface.
// Calling Marshal() marshals and returns the collection as well as an error.
func (c *Collection) Marshal() ([]byte, error) {
	response := make([]byte, 0)

	c.Documents.ForEach(func(key string, value interface{}) { // Iterate over skip list
		documentData, err := json.Marshal(value.(*Document)) // Type assert to *Document
		if err != nil {
			return
		}
		response = append(response, documentData...)
	})

	return response, nil
}
