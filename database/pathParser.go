package database

import (
	"fmt"
	"net/url"
	"strings"
)

// PathItem represents an item in the path (document or collection)
// Used to efficiently loop through to a point in the path
type PathItem interface {
	GetChildByName(name string) (PathItem, bool)
	Marshal() ([]byte, error)
}

// SplitPath splits the given path into its components.
func splitPath(path string) ([]string, error) {
	// Remove leading and trailing slashes if they exist.
	trimmedPath := strings.Trim(path, "/")

	// Split the path by slashes.
	parts := strings.Split(trimmedPath, "/")

	// Loop over parts to translate percent-encoded characters.
	for i, part := range parts {
		decodedPart, err := url.QueryUnescape(part)
		if err != nil {
			return nil, fmt.Errorf("Error decoding path part: %s", err)
		}
		parts[i] = decodedPart
	}

	// Remove the v1 from the path parts as we will not use it
	if len(parts) > 0 && parts[0] == "v1" {
		parts = parts[1:]
	}

	// Check if the path is empty
	if len(parts) == 0 {
		return nil, fmt.Errorf("Empty Path")
	}

	// The returned slice removes the leading and trailing slashes and decodes any percent-encoded values.
	return parts, nil
}
