// Package response writes mailbridge's small JSON response surface.
package response

import (
	"encoding/json"
	"net/http"
)

// Problem writes a JSON error response.
func Problem(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}

// JSON writes a JSON response with the supplied status code.
func JSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
