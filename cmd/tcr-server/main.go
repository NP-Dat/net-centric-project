package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/NP-Dat/net-centric-project/internal/server"
)

func main() {
	// Command line flags
	host := flag.String("host", "localhost", "Host to listen on")
	port := flag.Int("port", 8080, "Port to listen on")
	basePath := flag.String("basePath", getDefaultBasePath(), "Base path for config and data files")

	flag.Parse()

	// Create and start the server
	srv := server.NewServer(*host, *port, *basePath)
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	if err := srv.Stop(); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server stopped")
}

// getDefaultBasePath returns the default base path for config and data files
func getDefaultBasePath() string {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Printf("Warning: Failed to get current working directory: %v", err)
		return "."
	}

	// Check if we're in the root directory of the project
	if _, err := os.Stat(filepath.Join(cwd, "config", "towers.json")); err == nil {
		return cwd
	}

	// If we're in cmd/tcr-server, go up two levels
	if _, err := os.Stat(filepath.Join(cwd, "..", "..", "config", "towers.json")); err == nil {
		return filepath.Join(cwd, "..", "..")
	}

	// Default to current directory if not found
	log.Printf("Warning: Could not find config directory, using current directory")
	return cwd
}
