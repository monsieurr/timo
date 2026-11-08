package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"timo/internal/timo"
)

func main() {
	path, err := timo.DefaultPath()
	if err != nil {
		fail(err)
	}
	store := timo.NewStore(path)
	args := os.Args[1:]
	if desktopBuild == "true" && len(args) == 0 {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		if err := runDesktop(ctx, store); err != nil {
			desktopFailure(err)
		}
		return
	}
	if len(args) == 0 {
		if err := runREPL(store); err != nil {
			fail(err)
		}
		return
	}
	switch args[0] {
	case "stop-desktop":
		if err := stopDesktop(); err != nil {
			fail(err)
		}
	case "desktop", "app":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		if err := runDesktop(ctx, store); err != nil {
			fail(err)
		}
	case "web":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		if err := webCommand(ctx, store, args[1:]); err != nil {
			fail(err)
		}
	case "version", "--version":
		fmt.Println("timo 1.0.4")
	case "help", "--help", "-h":
		printHelp()
	case "where", "data":
		fmt.Println(store.Path)
	case "stats", "status", "settings", "set", "start", "pause", "resume", "reset", "stop", "skip", "focus", "break", "long_break":
		if err := runCommand(store, args); err != nil {
			fail(err)
		}
	default:
		fail(fmt.Errorf("unknown subcommand %q: run timo help", args[0]))
	}
}
func fail(err error) { fmt.Fprintln(os.Stderr, "timo:", err); os.Exit(1) }
