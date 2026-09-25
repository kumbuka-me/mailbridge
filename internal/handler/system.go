// Package handler contains mailbridge's HTTP endpoints.
package handler

import (
	"net/http"

	"github.com/containeroo/mailbridge/internal/response"
)

// Health reports that the process is ready to accept mail requests.
func Health() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// Readyz returns a lightweight readiness endpoint.
func Readyz() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// Version returns the running build version and commit.
func Version(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"version": version})
	}
}
