// Or should userList map username to UserStruct then store all the user + token info in UserStruct?

package authorization

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// Define constants for token length and character set
const strlen = 15
const charset = "AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz0123456789"

// Initialize a random number generator with a time-based seed
var seed = rand.New(rand.NewSource(time.Now().UnixNano()))

// Mutex for synchronizing access to tokenStore
var mu sync.Mutex

// Function to generate a random token
func makeToken() string {
	token := make([]byte, strlen) // Initialize a byte array to hold the token
	for i := range token {
		token[i] = charset[seed.Intn(len(charset))] // Populate token with random characters from charset
	}
	return string(token) // Convert byte array to string and return
}

// Struct to hold token information
type TokenInfo struct {
	Username string
	Created  time.Time
}

// Map to store token information
var tokenStore = make(map[string]TokenInfo) // map token to TokenInfo struct (username + time)

// HTTP handler function for authentication
func AuthHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		// For the /auth endpoint, indicate that POST and DELETE are allowed.
		w.Header().Set("Allow", "POST, DELETE")
		w.WriteHeader(http.StatusOK)
	case "POST": // Handle POST method for user authentication
		username := r.URL.Query().Get("username") // Get username from the query parameter
		if username == "" {
			http.Error(w, "Username is required", http.StatusBadRequest) // Return error if username is missing
			return
		}
		// ALSO NEED TO CHECK if user exists in the database here? or are all names valid?

		mu.Lock()                                                              // Lock the mutex to avoid race conditions
		token := makeToken()                                                   // Generate a new token
		tokenStore[token] = TokenInfo{Username: username, Created: time.Now()} // Store the token and other info
		mu.Unlock()                                                            // Unlock the mutex

		// Respond with the generated token
		w.Write([]byte(fmt.Sprintf("Logged in. Token: %s", token)))

	case "DELETE": // Handle DELETE method for user de-authentication
		token := r.Header.Get("Authorization")[7:] // to get the token after "Bearer "
		// Get token from the Authorization header
		if token == "" {
			http.Error(w, "Token is required", http.StatusBadRequest) // Return error if token is missing
			return
		}
		mu.Lock()                                      // Lock the mutex to avoid race conditions
		if info, exists := tokenStore[token]; exists { // Check if token exists
			if time.Since(info.Created).Hours() >= 1 { // Check token expiration
				delete(tokenStore, token)                               // Remove expired token
				http.Error(w, "Token expired", http.StatusUnauthorized) // Return an expiration error
				mu.Unlock()                                             // Unlock the mutex
				return
			}
		} else {
			http.Error(w, "Invalid token", http.StatusUnauthorized) // Return an error for invalid token
			mu.Unlock()                                             // Unlock the mutex
			return
		}
		delete(tokenStore, token) // Delete token if all checks pass
		mu.Unlock()               // Unlock the mutex

		w.Write([]byte("Logged out")) // Send logout confirmation

	default: // Handle unsupported HTTP methods
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// need this case in NewHandler() in main.go
// http.HandleFunc("/auth", authorization.authHandler)  // Route /auth URL path to authHandler function if /auth in URL
// need to do OPTIONS ad well
// Use LOGGING
// need to check for token expiration each time for all incoming requests with the token in the header
// UserStruct with token and username
