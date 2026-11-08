package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"timo/internal/timo"
)

//go:embed web/*
var webAssets embed.FS

type api struct{ store *timo.Store }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func requestJSON(r *http.Request, dest any) error {
	if r.Header.Get("Content-Type") != "application/json" {
		return errors.New("Content-Type must be application/json")
	}
	d := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 8192))
	d.DisallowUnknownFields()
	if err := d.Decode(dest); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("only one JSON object is allowed")
	}
	return nil
}
func (a api) state(w http.ResponseWriter, r *http.Request) {
	v, err := a.store.Snapshot(time.Now())
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, v)
}
func (a api) mutate(w http.ResponseWriter, r *http.Request) {
	if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+r.Host {
		writeJSON(w, 403, map[string]string{"error": "invalid origin"})
		return
	}
	if site := r.Header.Get("Sec-Fetch-Site"); site == "cross-site" {
		writeJSON(w, 403, map[string]string{"error": "cross-site request blocked"})
		return
	}
	var v timo.State
	var err error
	switch r.URL.Path {
	case "/api/action":
		var req struct {
			Action string `json:"action"`
		}
		if err = requestJSON(r, &req); err == nil {
			v, err = a.store.Update(time.Now(), func(s *timo.State) error { return s.Action(req.Action, time.Now()) })
		}
	case "/api/settings":
		var req struct {
			timo.Settings
			ExpectedSettingsRevision int64 `json:"expected_settings_revision"`
		}
		if err = requestJSON(r, &req); err == nil {
			v, err = a.store.Update(time.Now(), func(s *timo.State) error {
				if s.SettingsRevision != req.ExpectedSettingsRevision {
					return errors.New("settings changed in another window; reload and try again")
				}
				return s.SetSettings(req.Settings)
			})
		}
	default:
		writeJSON(w, 404, map[string]string{"error": "not found"})
		return
	}
	if err != nil {
		status := 400
		if strings.Contains(err.Error(), "settings changed") || strings.Contains(err.Error(), "timer is not") {
			status = 409
		}
		if strings.Contains(err.Error(), "read data") || strings.Contains(err.Error(), "replace JSON") || strings.Contains(err.Error(), "invalid data in") {
			status = 500
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, v)
}
func handler(store *timo.Store) http.Handler {
	a := api{store}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/state", a.state)
	mux.HandleFunc("POST /api/action", a.mutate)
	mux.HandleFunc("POST /api/settings", a.mutate)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		if name != "index.html" && name != "style.css" && name != "app.js" && name != "grain-dark.webp" && name != "grain-light.webp" {
			http.NotFound(w, r)
			return
		}
		b, err := webAssets.ReadFile("web/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		switch name {
		case "index.html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		case "style.css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case "app.js":
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		case "grain-dark.webp", "grain-light.webp":
			w.Header().Set("Content-Type", "image/webp")
		}
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("GET /api/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 404, map[string]string{"error": "not found"})
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Reject DNS rebinding: the browser may only address this local application
		// through a loopback hostname, even when Docker binds 0.0.0.0 internally.
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if host != "127.0.0.1" && host != "localhost" {
			http.Error(w, "local host required", http.StatusForbidden)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Timo-App", "1")
		mux.ServeHTTP(w, r)
	})
}
func launchBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
func serve(ctx context.Context, store *timo.Store, host string, port int, open bool) error {
	return serveWithMode(ctx, store, host, port, open, false)
}

func serveWithMode(ctx context.Context, store *timo.Store, host string, port int, open bool, desktop bool) error {
	if port < 1 || port > 65535 {
		return errors.New("port must be 1–65535")
	}
	if host != "127.0.0.1" && host != "localhost" && host != "0.0.0.0" {
		return errors.New("host must be 127.0.0.1, localhost, or 0.0.0.0")
	}
	if _, err := store.Snapshot(time.Now()); err != nil {
		return err
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("cannot listen on %s: %w", addr, err)
	}
	appHandler := handler(store)
	if desktop {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
		mux := http.NewServeMux()
		mux.HandleFunc("POST /api/desktop/shutdown", func(w http.ResponseWriter, r *http.Request) {
			if r.Host != "127.0.0.1:8765" && r.Host != "localhost:8765" ||
				r.Header.Get("X-Timo-Control") != "shutdown" || r.Header.Get("Origin") != "" {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			go cancel()
		})
		mux.Handle("/", appHandler)
		appHandler = mux
	}
	srv := &http.Server{Handler: appHandler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second}
	url := "http://127.0.0.1:" + strconv.Itoa(port)
	fmt.Printf("Browser dashboard: %s\nPress Ctrl+C to stop the web server.\n", url)
	if open {
		if err := launchBrowser(url); err != nil {
			log.Printf("could not open browser: %v (open %s manually)", err, url)
		}
	}
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			timeout, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = srv.Shutdown(timeout)
		case <-done:
		}
	}()
	err = srv.Serve(ln)
	close(done)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
func webCommand(ctx context.Context, store *timo.Store, args []string) error {
	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	port := fs.Int("port", 8765, "local HTTP port")
	host := fs.String("host", "127.0.0.1", "bind interface (0.0.0.0 only for isolated Docker use)")
	noOpen := fs.Bool("no-open", false, "do not open a browser automatically")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return serve(ctx, store, *host, *port, !(*noOpen))
}
