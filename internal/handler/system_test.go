package handler

func TestAPI_Healthz(t *testing.T) {
	t.Parallel()
	t.Run("returns ok", func(t *testing.T) {
		t.Parallel()

		api, _ := testAPI("", testLogger())
		rec := httptest.NewRecorder()

		api.Healthz().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))
		assert.Equal(t, "ok", rec.Body.String())
	})
}
