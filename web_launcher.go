package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// The REPL starts the web UI in-process; closing the REPL closes this server.
// `timo web` starts an independent foreground server for browser-only use.
func (t *terminal) openWeb() error {
	const url = "http://127.0.0.1:8765"
	if t.webServer == nil {
		ln, err := net.Listen("tcp", "127.0.0.1:8765")
		if err != nil {
			client := &http.Client{Timeout: 600 * time.Millisecond}
			resp, reqErr := client.Get(url + "/api/state")
			if reqErr != nil {
				return fmt.Errorf("port 8765 is unavailable: %w", err)
			}
			defer resp.Body.Close()
			if resp.Header.Get("X-Timo-App") != "1" || resp.StatusCode != 200 {
				return fmt.Errorf("port 8765 is in use by another service")
			}
		} else {
			srv := &http.Server{Handler: handler(t.store), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second}
			t.webServer = srv
			go func() { _ = srv.Serve(ln) }()
		}
	}
	if err := launchBrowser(url); err != nil {
		return fmt.Errorf("open %s manually: %w", url, err)
	}
	fmt.Println("Dashboard:", url)
	if t.webServer != nil {
		fmt.Println("The web server stays available until you quit this REPL.")
	}
	return nil
}
func (t *terminal) closeWeb() {
	if t.webServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = t.webServer.Shutdown(ctx)
	}
}
