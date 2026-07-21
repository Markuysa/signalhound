// Command signalhound is the single binary: agent, API, and embedded UI.
//
// The foundation ticket lands only the skeleton — flag parsing and config loading — so
// the module builds and later tickets have an entrypoint to wire into. serve and demo
// are stubs until the app composition root (issue #19) exists.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Markuysa/signalhound/internal/config"
)

var version = "0.1.0-dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "serve":
		os.Exit(serve(os.Args[2:]))
	case "demo":
		os.Exit(demo(os.Args[2:]))
	case "version", "-v", "--version":
		fmt.Println(version)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `signalhound %s

usage:
  signalhound serve [-config path]   run the agent, API and UI
  signalhound demo                   run with seed data, no external calls
  signalhound version                print the version
`, version)
}

func serve(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	path := fs.String("config", "config.yaml", "path to config.yaml")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := config.Load(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	// The composition root (issue #19) turns this config into a running agent. Until it
	// lands, prove the config parsed and stop rather than pretending to serve.
	fmt.Printf("config OK: provider=%s model=%s sources listen=%s\n",
		cfg.LLM.Provider, cfg.LLM.Model, cfg.Server.Addr)
	fmt.Fprintln(os.Stderr, "serve is not wired yet (blocked on the composition root, issue #19)")
	return 0
}

func demo(_ []string) int {
	fmt.Fprintln(os.Stderr, "demo mode is not wired yet (issue #13)")
	return 0
}
