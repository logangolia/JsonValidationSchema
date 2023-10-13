package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/RICE-COMP318-FALL23/owldb-p1group37/skiplist"
)

// The Collection struct represents a collection in a database.
type Collection struct {
	Name      string                               `json:"-"`
	Documents skiplist.SkipList[string, *Document] `json:"-"`
	URI       string                               `json:"uri"`
}

// NewCollection creates and returns a new Collection struct with the given name.
func NewCollection(name string, uri string) *Collection {
	return &Collection{
		Name:      name,
		Documents: skiplist.NewSkipList[string, *Document](),
		URI:       uri,
	}
}

// GetChildByName implements the function from the PathItem interface.
// If it exists, it returns the document and true, otherwise nil and false.
func (c *Collection) GetChildByName(name string) (PathItem, bool) {
	child, exists := c.Documents.Find(name)
	if exists {
		return child, true
	}
	return nil, false
}

// Marshal implements the function from the PathItem interface.
// Calling Marshal() marshals and returns the collection as well as an error.
func (c *Collection) Marshal() ([]byte, error) {
	ctx := context.TODO()

	// Query for nodes
	documentPairs, err := c.Documents.Query(ctx,
		c.Documents.(*skiplist.SkipListImpl[string, *Document]).Head.Pair.Key,
		c.Documents.(*skiplist.SkipListImpl[string, *Document]).Tail.Pair.Key)
	if err != nil {
		return nil, err
	}

	var documents []*Document
	// Append each document to the slice
	for _, pair := range documentPairs {
		documents = append(documents, pair.Value)
	}

	// Marshal the entire slice into its JSON representation
	return json.Marshal(documents)
}

// MarshalURI is a function that marshals the collection itself, rather than the documents inside it
// MarshalURI is used in response to PUT for collections, while Marshal is used on GET
func (c *Collection) MarshalURI() ([]byte, error) {
	response, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("Error marshaling dollection: %w", err)
	}
	return response, nil
}
