package main

import (
	"flag"
	"fmt"
	"log"

	"ai-proxy/internal/server"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// CLI flags
	showVersion := flag.Bool("version", false, "print version and exit")
	configPath := flag.String("config", "", "path to provider-config.yaml (required)")
	envFilePath := flag.String("env-file", "", "path to .env file")
	addr := flag.String("addr", "", "listen address override, e.g., :8080")
	flag.Parse()

	if *showVersion {
		fmt.Printf("ai-proxy %s (commit %s, built %s)\n", version, commit, date)
		return
	}

	// Create and start the server
	srv, err := server.NewServerFromConfig(*configPath, *envFilePath, *addr)
	if err != nil {
		log.Fatal(err.Error())
	}

	if err := srv.Start(); err != nil {
		log.Fatal(err.Error())
	}
}
