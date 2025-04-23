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

		// Use the ParseCommand function from the client package instead of local parsing
		err = c.ParseCommand(input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}

// printHelp shows available commands
func printHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  login - Login to the server interactively")
	fmt.Println("  join - Join the matchmaking queue")
	fmt.Println("  send <message> - Send a message to the server")
	fmt.Println("  quit/exit - Disconnect and quit")
	fmt.Println("  help - Show this help message")
}
