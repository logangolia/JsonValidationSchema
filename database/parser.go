package database

import (
	"fmt"
	"net/url"
	"strings"
)

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
