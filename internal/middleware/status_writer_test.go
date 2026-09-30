package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStatusWriterInformationalResponse(t *testing.T) {
	finalStatus := make(chan int, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := &statusWriter{ResponseWriter: w}
		response.WriteHeader(http.StatusEarlyHints)
		response.WriteHeader(http.StatusAccepted)
		response.WriteHeader(http.StatusInternalServerError)
		finalStatus <- response.status
	}))
	defer server.Close()
	response, err := server.Client().Get(server.URL)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusAccepted, response.StatusCode)
	require.Equal(t, http.StatusAccepted, <-finalStatus)
}

func TestStatusWriterResponseController(t *testing.T) {
	recorder := httptest.NewRecorder()
	writer := &statusWriter{ResponseWriter: recorder}
	require.NoError(t, http.NewResponseController(writer).Flush())
	require.True(t, recorder.Flushed)
}
