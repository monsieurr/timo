package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"timo/internal/timo"
)

// desktopBuild is set to "true" only on the double-clickable desktop binary.
// The normal CLI stays a console program and retains the REPL as its default.
var desktopBuild = "false"

const desktopURL = "http://127.0.0.1:8765"

// timoServerAt returns true only for Timo's own loopback server. A different
// service on the port must never be mistaken for an existing Timo instance.
func timoServerAt(url string) bool {
	client := &http.Client{Timeout: 600 * time.Millisecond}
	resp, err := client.Get(url + "/api/state")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK && resp.Header.Get("X-Timo-App") == "1"
}

// runDesktop starts a background-friendly browser host. Repeated launches
// reopen the existing dashboard instead of competing for port 8765.
func runDesktop(ctx context.Context, store *timo.Store) error {
	if timoServerAt(desktopURL) {
		return launchBrowser(desktopURL)
	}
	err := serveWithMode(ctx, store, "127.0.0.1", 8765, true, true)
	if err == nil {
		return nil
	}
	// Two launchers might race: one can bind immediately after the probe.
	for i := 0; i < 6; i++ {
		if timoServerAt(desktopURL) {
			return launchBrowser(desktopURL)
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("start desktop dashboard: %w", err)
}

// stopDesktop stops only the background desktop host, not a REPL-owned web server.
func stopDesktop() error {
	req, err := http.NewRequest(http.MethodPost, desktopURL+"/api/desktop/shutdown", nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Timo-Control", "shutdown")
	client := &http.Client{Timeout: 750 * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("desktop host is not running: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("desktop host did not accept shutdown (HTTP %d)", resp.StatusCode)
	}
	return nil
}
