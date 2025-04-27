package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/NP-Dat/net-centric-project/internal/client"
	"github.com/NP-Dat/net-centric-project/pkg/logger"
)

func main() {
	// Command line flags
	host := flag.String("host", "localhost", "Server host to connect to")
	port := flag.Int("port", 8080, "Server port to connect to")
	logLevel := flag.String("logLevel", "info", "Log level (debug, info, warn, error)")

	flag.Parse()

	// Initialize logging system
	initLogging(*logLevel)

	logger.Client.Info("Text Clash Royale Client starting")
	logger.Client.Info("Connecting to server at %s:%d", *host, *port)

	// Create and connect client
	c := client.NewClient(*host, *port)
	c.SetupDefaultHandlers()

	fmt.Printf("Connecting to server at %s:%d...\n", *host, *port)
	err := c.Connect()
	if err != nil {
		logger.Client.Fatal("Failed to connect to server: %v", err)
	}
	fmt.Println("Connected to server!")

	// Start CLI loop in a goroutine
	go cliLoop(c)

	// Wait for disconnect
	c.WaitForDisconnect()
	fmt.Println("Disconnected from server.")
	logger.Client.Info("Client disconnected")
}

// cliLoop runs the command line interface loop
func cliLoop(c *client.Client) {
	reader := bufio.NewReader(os.Stdin)

	for c.IsConnected() {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			logger.Client.Error("Error reading input: %v", err)
			fmt.Printf("Error reading input: %v\n", err)
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle special debug command for changing log level dynamically
		if strings.HasPrefix(input, "debug loglevel ") {
			parts := strings.Fields(input)
			if len(parts) == 3 {
				setLogLevel(parts[2])
				continue
			}
		}

		// Use the ParseCommand function from the client package
		err = c.ParseCommand(input)
		if err != nil {
			logger.Client.Warn("Command failed: %v", err)
			fmt.Printf("Error: %v\n", err)
		}
	}
}

// initLogging initializes the logging system
func initLogging(logLevelStr string) {
	// Get the user's home directory for logs
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Warning: Failed to determine user home directory: %v", err)
		homeDir = "."
	}

	// Create logs directory in the home directory or current directory
	logsDir := filepath.Join(homeDir, ".tcr", "logs")
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
	setLogLevel(logLevelStr)
}

// setLogLevel dynamically changes the log level
func setLogLevel(levelStr string) {
	var logLevel logger.LogLevel
	switch strings.ToLower(levelStr) {
	case "debug":
		logLevel = logger.DEBUG
		fmt.Println("Log level set to DEBUG")
	case "info":
		logLevel = logger.INFO
		fmt.Println("Log level set to INFO")
	case "warn":
		logLevel = logger.WARN
		fmt.Println("Log level set to WARN")
	case "error":
		logLevel = logger.ERROR
		fmt.Println("Log level set to ERROR")
	default:
		fmt.Printf("Unknown log level: %s, using INFO\n", levelStr)
		logLevel = logger.INFO
	}

	logger.SetGlobalLogLevel(logLevel)
}
