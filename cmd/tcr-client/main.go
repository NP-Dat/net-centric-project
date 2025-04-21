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

	// Don't prompt for login automatically - wait for the user to use the login command
	// or respond to server events (like login prompts)

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
			// Use our interactive prompt
			err := c.PromptLogin()
			if err != nil {
				fmt.Printf("Failed to login: %v\n", err)
			}

		case "join":
			// Join the matchmaking queue
			err := c.JoinMatchmaking()
			if err != nil {
				fmt.Printf("Failed to join matchmaking: %v\n", err)
			} else {
				fmt.Println("Joining matchmaking queue...")
			}

		case "send":
			if len(parts) < 2 {
				fmt.Println("Usage: send <message>")
				continue
			}
			message := strings.Join(parts[1:], " ")
			err := c.SendMessage(message)
			if err != nil {
				fmt.Printf("Failed to send message: %v\n", err)
			} else {
				fmt.Println("Message sent")
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
	fmt.Println("  login - Login to the server interactively")
	fmt.Println("  join - Join the matchmaking queue")
	fmt.Println("  send <message> - Send a message to the server")
	fmt.Println("  quit/exit - Disconnect and quit")
	fmt.Println("  help - Show this help message")
}
