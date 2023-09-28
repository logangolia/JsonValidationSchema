package database

import (
	"encoding/json"
	"fmt"
)

type Collection struct {
	Name      string               `json:"-"`
	Documents map[string]*Document `json:"-"`
	URI       string               `json:"uri"`
}

func NewCollection(name string) *Collection {
	return &Collection{
		Name:      name,
		Documents: make(map[string]*Document),
		URI:       name,
	}
}

func marshalCollection(collection *Collection) ([]byte, error) {
	response := make([]byte, 0)

	for _, document := range collection.Documents {
		documentData, err := json.Marshal(document)
		if err != nil {
			return nil, fmt.Errorf("Error marshaling document: %w", err)
		}
		response = append(response, documentData...)
	}

	return response, nil
}
