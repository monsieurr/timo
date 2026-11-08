package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"timo/internal/timo"
)

func doAPI(h http.Handler, method, path, body, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "http://127.0.0.1:8765"+path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}
func TestAPIEndToEndAndOrigin(t *testing.T) {
	h := handler(timo.NewStore(filepath.Join(t.TempDir(), "data.json")))
	initial := doAPI(h, "GET", "/api/state", "", "")
	if initial.Code != 200 || initial.Header().Get("X-Timo-App") != "1" {
		t.Fatalf("initial: %d", initial.Code)
	}
	rebinding := httptest.NewRequest("GET", "http://attacker.example:8765/api/state", nil)
	badHost := httptest.NewRecorder()
	h.ServeHTTP(badHost, rebinding)
	if badHost.Code != 403 {
		t.Fatalf("DNS rebinding host accepted: %d", badHost.Code)
	}
	blocked := doAPI(h, "POST", "/api/action", `{"action":"start"}`, "https://evil.example")
	if blocked.Code != 403 {
		t.Fatalf("CSRF was not blocked: %d", blocked.Code)
	}
	malformed := doAPI(h, "POST", "/api/action", `{"action":"start","extra":4}`, "")
	if malformed.Code != 400 {
		t.Fatalf("unknown input was not rejected: %d", malformed.Code)
	}
	started := doAPI(h, "POST", "/api/action", `{"action":"start"}`, "http://127.0.0.1:8765")
	if started.Code != 200 {
		t.Fatalf("start: %d %s", started.Code, started.Body.String())
	}
	var v timo.State
	if err := json.Unmarshal(started.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	if v.Timer.Status != "running" {
		t.Fatal("timer not running")
	}
	conflict := doAPI(h, "POST", "/api/action", `{"action":"start"}`, "")
	if conflict.Code != 409 {
		t.Fatalf("duplicate start: %d", conflict.Code)
	}
	// Terminal and HTTP mutations use the same store and see each other.
	cfg := doAPI(h, "POST", "/api/settings", `{"focus_minutes":31,"break_minutes":5,"long_break_minutes":15,"long_break_every":4,"accent":"#AABBCC","expected_settings_revision":0}`, "")
	if cfg.Code != 200 {
		t.Fatalf("settings: %d %s", cfg.Code, cfg.Body.String())
	}
	stale := doAPI(h, "POST", "/api/settings", `{"focus_minutes":35,"break_minutes":5,"long_break_minutes":15,"long_break_every":4,"accent":"#AABBCC","expected_settings_revision":0}`, "")
	if stale.Code != 409 {
		t.Fatalf("stale settings should conflict, got %d", stale.Code)
	}
	paused := doAPI(h, "POST", "/api/action", `{"action":"pause"}`, "")
	if paused.Code != 200 {
		t.Fatalf("pause: %d %s", paused.Code, paused.Body.String())
	}
	s, err := timo.NewStore(filepath.Join(t.TempDir(), "different.json")).Snapshot(time.Now())
	if err != nil || s.Settings.FocusMinutes != 25 {
		t.Fatal("stores should be independent")
	}
}
func TestServedAssets(t *testing.T) {
	h := handler(timo.NewStore(filepath.Join(t.TempDir(), "data.json")))
	for _, p := range []string{"/", "/style.css", "/app.js", "/grain-dark.webp", "/grain-light.webp"} {
		res := doAPI(h, "GET", p, "", "")
		if res.Code != 200 || res.Body.Len() < 100 {
			t.Errorf("asset %s: %d", p, res.Code)
		}
	}
	for _, p := range []string{"/config.json", "/../main.go"} {
		res := doAPI(h, "GET", p, "", "")
		if res.Code != 404 && res.Code != 301 {
			t.Errorf("unexpected asset exposed: %s", p)
		}
	}
}
