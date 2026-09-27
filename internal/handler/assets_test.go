package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssets(t *testing.T) {
	t.Parallel()

	assets := fstest.MapFS{
		"favicon.svg": {
			Data: []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`),
		},
		"index.gohtml": {
			Data: []byte(`<html></html>`),
		},
		"icons/icon.txt": {
			Data: []byte("asset"),
		},
	}

	t.Run("serves embedded asset", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()
		Assets(assets).ServeHTTP(
			response,
			httptest.NewRequest(http.MethodGet, "/assets/favicon.svg", nil),
		)

		require.Equal(t, http.StatusOK, response.Code)
		assert.Equal(t, "image/svg+xml", response.Header().Get("Content-Type"))
		assert.Equal(t, "public, max-age=3600", response.Header().Get("Cache-Control"))
		assert.Contains(t, response.Body.String(), "<svg")
	})

	t.Run("serves nested asset", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()
		Assets(assets).ServeHTTP(
			response,
			httptest.NewRequest(http.MethodGet, "/assets/icons/icon.txt", nil),
		)

		require.Equal(t, http.StatusOK, response.Code)
		assert.Equal(t, "asset", response.Body.String())
	})

	t.Run("does not expose templates", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()
		Assets(assets).ServeHTTP(
			response,
			httptest.NewRequest(http.MethodGet, "/assets/index.gohtml", nil),
		)

		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("does not list directories", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()
		Assets(assets).ServeHTTP(
			response,
			httptest.NewRequest(http.MethodGet, "/assets/", nil),
		)

		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("missing asset", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()
		Assets(assets).ServeHTTP(
			response,
			httptest.NewRequest(http.MethodGet, "/assets/missing.svg", nil),
		)

		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("missing filesystem", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()
		Assets(nil).ServeHTTP(
			response,
			httptest.NewRequest(http.MethodGet, "/assets/favicon.svg", nil),
		)

		assert.Equal(t, http.StatusNotFound, response.Code)
	})
}

func TestPublicAssetPath(t *testing.T) {
	t.Parallel()

	assert.True(t, publicAssetPath("favicon.svg"))
	assert.True(t, publicAssetPath("icons/favicon.png"))
	assert.False(t, publicAssetPath(""))
	assert.False(t, publicAssetPath("../favicon.svg"))
	assert.False(t, publicAssetPath("index.gohtml"))
}
