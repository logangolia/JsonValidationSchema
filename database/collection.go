package database

import (
	"bytes"
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
	var buffer bytes.Buffer
	ctx := context.TODO()

	// Type assertion
	impl, ok := c.Documents.(*skiplist.SkipListImpl[string, *Document])
	if !ok {
		return nil, fmt.Errorf("Documents is not an instance of SkipListImpl")
	}

	// Query for nodes
	documentPairs, err := c.Documents.Query(ctx, impl.Head.Pair.Key, impl.Tail.Pair.Key)
	if err != nil {
		return nil, err
	}

	fmt.Println("documentPairs")
	fmt.Println(documentPairs)

	// Marshal nodes into the buffer
	for i := range documentPairs {
		docBytes, err := json.Marshal(documentPairs[i].Value)
		if err != nil {
			return nil, err
		}
		buffer.Write(docBytes)
	}

	fmt.Println("buffer.Bytes()")
	fmt.Println(buffer.Bytes())

	return buffer.Bytes(), nil
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
