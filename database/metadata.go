package database

import (
	"time"
)

// The Metadata struct holds the metadata of a given document.
type Metadata struct {
	CreatedBy      string
	CreatedAt      time.Time
	LastModifiedBy string
	LastModifiedAt time.Time
}

// NewMetadata creates and returns a new Metadata struct based on the inputs.
func NewMetadata(user string, time time.Time) *Metadata {
	return &Metadata{
		CreatedBy:      "server",
		CreatedAt:      time,
		LastModifiedBy: "server",
		LastModifiedAt: time,
	}
}
