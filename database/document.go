package database

import (
	"encoding/json"
	"fmt"
	"time"
)

type Document struct {
	Name        string `json:"path"`
	Data        []byte `json:"doc"`
	Collections map[string]*Collection
	Metadata    Metadata `json:"meta"`
	URI         string   `json:"uri"`
}

type Metadata struct {
	CreatedBy      string
	CreatedAt      time.Time
	LastModifiedBy string
	LastModifiedAt time.Time
}

func NewDocument(name string, data []byte, user string, time time.Time, uriPrefix string) *Document {
	return &Document{
		Name:        name,
		Data:        data,
		Collections: make(map[string]*Collection),
		Metadata:    *NewMetadata(user, time),
		URI:         uriPrefix + name,
	}
}

func NewMetadata(user string, time time.Time) *Metadata {
	return &Metadata{
		CreatedBy:      "server",
		CreatedAt:      time,
		LastModifiedBy: "server",
		LastModifiedAt: time,
	}
}

func marshalDocument(document *Document) ([]byte, error) {
	response, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("Error marshaling document: %w", err)
	}
	return response, nil
}
