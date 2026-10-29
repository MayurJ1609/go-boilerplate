package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoggingPreservesResponse(t *testing.T) {
	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("accepted"))
	}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusAccepted)
	}
	if got, want := recorder.Body.String(), "accepted"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
