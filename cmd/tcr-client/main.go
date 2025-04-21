package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/NP-Dat/net-centric-project/internal/client"
)

func main() {
	// Command line flags
	host := flag.String("host", "localhost", "Server host to connect to")
	port := flag.Int("port", 8080, "Server port to connect to")

	flag.Parse()

	// Create and connect client
	c := client.NewClient(*host, *port)
	c.SetupDefaultHandlers()

	fmt.Printf("Connecting to server at %s:%d...\n", *host, *port)
	err := c.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	fmt.Println("Connected to server!")

	// Start CLI loop in a goroutine
	go cliLoop(c)

	// Wait for disconnect
	c.WaitForDisconnect()
	fmt.Println("Disconnected from server.")
}

// cliLoop runs the command line interface loop
func cliLoop(c *client.Client) {
	reader := bufio.NewReader(os.Stdin)

	for c.IsConnected() {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := parts[0]

		switch strings.ToLower(command) {
		case "login":
			if len(parts) < 3 {
				fmt.Println("Usage: login <username> <password>")
				continue
			}
			username := parts[1]
			password := parts[2]

			err := c.LoginWithCredentials(username, password)
			if err != nil {
				fmt.Printf("Failed to login: %v\n", err)
			} else {
				fmt.Println("Login request sent")
			}

		case "quit", "exit":
			fmt.Println("Disconnecting from server...")
			c.Disconnect()
			return

		case "help":
			printHelp()

		default:
			fmt.Printf("Unknown command: %s\n", command)
			fmt.Println("Type 'help' to see available commands")
		}
	}
}

// printHelp shows available commands
func printHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  login <username> <password> - Login to the server")
	fmt.Println("  quit/exit - Disconnect and quit")
	fmt.Println("  help - Show this help message")
}
