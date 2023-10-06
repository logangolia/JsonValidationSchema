package database

import "github.com/RICE-COMP318-FALL23/owldb-p1group37/skiplist"

// The Collection struct represents a collection in a database.
type Collection struct {
	Name      string
	Documents skiplist.SkipList[string, Document]
	URI       string `json:"uri"`
}

// NewCollection creates and returns a new Collection struct with the given name.
func NewCollection(name string) *Collection {
	return &Collection{
		Name:      name,
		Documents: skiplist.NewSkipList[string, Document](),
		URI:       name,
	}
}

// GetChildByName implements the function from the PathItem interface.
// If it exists, it returns the document and true, otherwise nil and false.
func (c *Collection) GetChildByName(name string) (PathItem, bool) {
	child, exists := c.Documents.Find(name)
	if exists {
		return &child, true
	}
	return nil, false
}

// Marshal implements the function from the PathItem interface.
// Calling Marshal() marshals and returns the collection as well as an error.
func (c *Collection) Marshal() ([]byte, error) {
	response := make([]byte, 0)
	return response, nil
}
