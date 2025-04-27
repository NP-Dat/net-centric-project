package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/NP-Dat/net-centric-project/internal/server"
	"github.com/NP-Dat/net-centric-project/pkg/logger"
)

func main() {
	// Command line flags
	host := flag.String("host", "localhost", "Host to listen on")
	port := flag.Int("port", 8080, "Port to listen on")
	basePath := flag.String("basePath", getDefaultBasePath(), "Base path for config and data files")
	logLevel := flag.String("logLevel", "info", "Log level (debug, info, warn, error)")

	flag.Parse()

	// Initialize logging system
	initLogging(*basePath, *logLevel)

	// Log startup information
	logger.Server.Info("Server starting up...")
	logger.Server.Info("Host: %s, Port: %d", *host, *port)
	logger.Server.Info("Base path: %s", *basePath)

	// Create and start the server
	srv := server.NewServer(*host, *port, *basePath)
	if err := srv.Start(); err != nil {
		logger.Server.Fatal("Failed to start server: %v", err)
	}

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Server.Info("Received shutdown signal")

	if err := srv.Stop(); err != nil {
		logger.Server.Fatal("Server shutdown failed: %v", err)
	}

	logger.Server.Info("Server stopped gracefully")
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

// initLogging initializes the logging system
func initLogging(basePath string, logLevelStr string) {
	// Create logs directory if it doesn't exist
	logsDir := filepath.Join(basePath, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		log.Printf("Warning: Failed to create logs directory: %v", err)
		// Continue with console logging only
	} else {
		// Initialize file logging
		if err := logger.InitializeFileLogging(logsDir); err != nil {
			log.Printf("Warning: Failed to initialize file logging: %v", err)
			// Continue with console logging only
		}
	}

	// Set log level based on command line flag
	var logLevel logger.LogLevel
	switch logLevelStr {
	case "debug":
		logLevel = logger.DEBUG
	case "info":
		logLevel = logger.INFO
	case "warn":
		logLevel = logger.WARN
	case "error":
		logLevel = logger.ERROR
	default:
		fmt.Printf("Unknown log level: %s, using INFO\n", logLevelStr)
		logLevel = logger.INFO
	}

	logger.SetGlobalLogLevel(logLevel)
}
