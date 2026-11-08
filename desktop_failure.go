package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Desktop binaries may not have a console. Keep failures discoverable.
func desktopFailure(err error) {
	dir, dirErr := os.UserConfigDir()
	if dirErr != nil {
		return
	}
	path := filepath.Join(dir, "timo")
	if os.MkdirAll(path, 0700) != nil {
		return
	}
	f, openErr := os.OpenFile(filepath.Join(path, "launcher.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if openErr != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s: %v\n", time.Now().Format(time.RFC3339), err)
}
