package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTimoServerAt(t *testing.T) {
	for _, tc := range []struct {
		name     string
		marker   string
		status   int
		expected bool
	}{
		{"ours", "1", 200, true},
		{"foreign service", "", 200, false},
		{"foreign marker", "0", 200, false},
		{"server error", "1", 500, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/state" {
					t.Errorf("bad URL: %s", r.URL.Path)
				}
				if tc.marker != "" {
					w.Header().Set("X-Timo-App", tc.marker)
				}
				w.WriteHeader(tc.status)
			}))
			defer server.Close()
			if got := timoServerAt(server.URL); got != tc.expected {
				t.Errorf("got %v want %v", got, tc.expected)
			}
		})
	}
}
