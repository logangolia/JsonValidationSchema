// Or should userList map username to UserStruct then store all the user + token info in UserStruct?

package authorization

import (
	"encoding/json"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"time"
)

// Initialize a random number generator with a time-based seed
var seed = rand.New(rand.NewSource(time.Now().UnixNano()))

// Define parameters of token for generation:
// charset is a set of characters to choose to make the token out of
// tokenLen is the number of characters the token will be made of
// For a more secure authorization, use more characters
const charset = "AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz0123456789"
const tokenLen = 15

// authHandler struct, which contains operations which only act on /auth
type AuthHandler struct {
	tokenStore map[string]string
}

// userFormat to unmarshal user data into
type UserFormat struct {
	Username string
}

// creates a new authHandler with a token store initialized
func NewAuth() *AuthHandler {
	a := new(AuthHandler)
	a.tokenStore = make(map[string]string)
	return a
}

// Function to generate a random token
func (auth AuthHandler) makeToken() string {
	token := make([]byte, tokenLen) // Initialize a byte array to hold the token
	for i := range token {
		token[i] = charset[seed.Intn(len(charset))] // Populate token with random characters from charset
	}
	slog.Info("Token made" + string(token))
	return string(token) // Convert byte array to string and return
}

// HTTP handler function for authentication
func (auth AuthHandler) HandleAuthFunctions(w http.ResponseWriter, r *http.Request) {
	slog.Info("Auth Method Called ", r.Method)
	slog.Info("Path ", r.URL.Path)
	logHeader(r)

	//Switch between types of methods
	switch r.Method {
	case http.MethodPost: // Handle POST method for user authentication
		slog.Info("post request at /auth")
		auth.authPost(w, r)
		slog.Info("post finished")
	case http.MethodDelete: // Handle DELETE method for user de-authentication
		slog.Info("delete request at /auth")
		auth.authDelete(w, r)
		slog.Info("delete finished")
	case http.MethodOptions:
		slog.Info("auth requests options")
		auth.authOptions(w, r)
		slog.Info("options finished")
	default: // Handle unsupported HTTP methods
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

// Handles options request to /auth
// Writes header for preflight request
func (auth AuthHandler) authOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "POST,DELETE")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	slog.Info("Auth options header written")
	w.WriteHeader(http.StatusOK)
}

// Handles post request to /auth
func (auth *AuthHandler) authPost(w http.ResponseWriter, r *http.Request) {
	//Detect if content-type is application/json
	if r.Header.Get("Content-Type") != "" {
		content := r.Header.Get("Content-Type")
		if content != "application/json" {
			http.Error(w, "Content header not JSON", http.StatusUnsupportedMediaType)
			return
		}
	} else {
		slog.Info("Header contains no content type")
		return
	}

	//Read JSONValue from Request Header
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Info("Body could not be read")
		http.Error(w, `"invalid user format"`, http.StatusBadRequest)
		return
	}
	slog.Info("Read Body succeeded")
	r.Body.Close()

	//Umarshal data into userFormat struct, containing username: user
	var d UserFormat
	err = json.Unmarshal(body, &d)
	if err != nil {
		slog.Info("decode failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	slog.Info("Unmarshaled successfully")

	//Checks that a nonempty username was sent
	if d.Username == "" {
		slog.Info("No username")
		http.Error(w, "Username is required", http.StatusBadRequest) // Return error if username is missing
		return
	}

	slog.Info("Username exists")

	// ALSO NEED TO CHECK if user exists in the database here? or are all names valid?
	token := auth.makeToken() // Generate a new token
	time.AfterFunc(1*time.Hour, func() { delete(auth.tokenStore, token) })

	auth.tokenStore[token] = d.Username // Store the token and other info
	slog.Info(auth.tokenStore[token])
	// Respond with the generated token
	response := marshalToken(token)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func (auth *AuthHandler) authDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")

	//Checks user authorization
	if r.Header.Get("Authorization") == "" {
		w.Header().Add("WWW-Authenticate", "Bearer")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	token := r.Header.Get("Authorization")[7:]

	//Checks that token is in tokenStore
	if auth.CheckToken(r.Header.Get("Authorization")) != true {
		w.Header().Add("WWW-Authenticate", "Bearer")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//Deletes token
	delete(auth.tokenStore, token)
	slog.Info(auth.tokenStore[token])
	w.WriteHeader(http.StatusNoContent)
}

// Packs the token up to be sent back to the user.
func marshalToken(token string) []byte {
	tokenVal := map[string]string{"token": token}

	response, err := json.MarshalIndent(tokenVal, "", "  ")
	if err != nil {
		slog.Info("Token marshaling failed")
		return nil
	}
	return response
}

// simple function to check that the given token is in the tokenStore
func (auth *AuthHandler) CheckToken(token string) bool {
	slog.Info(token)
	for k, v := range auth.tokenStore {
		slog.Info(k, v)
		if token[7:] == k {
			if v != "" {
				return true
			}
		}
	}
	return false
}

// Reads the token file into tokenStore
func (auth *AuthHandler) handleTokenFile(path string) {
	dat, err := os.ReadFile(path)
	if err != nil {
		slog.Info("Token file could not be read")
		return
	}

	var tokens map[string]string
	err = json.Unmarshal(dat, &tokens)
	if err != nil {
		slog.Info("Unmarshal Failed")
		return
	}

	for user, token := range tokens {
		auth.tokenStore[token] = user
	}
}

func logHeader(r *http.Request) {
	for key, element := range r.Header {
		slog.Info("Header:", key, "Value", element)
	}
}

// need this case in NewHandler() in main.go
// http.HandleFunc("/auth", authorization.authHandler)  // Route /auth URL path to authHandler function if /auth in URL
// need to do OPTIONS ad well
// Use LOGGING
// need to check for token expiration each time for all incoming requests with the token in the header
// UserStruct with token and username
