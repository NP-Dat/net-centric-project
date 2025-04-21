package server

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/NP-Dat/net-centric-project/internal/models"
	"github.com/NP-Dat/net-centric-project/internal/network"
	"github.com/NP-Dat/net-centric-project/internal/persistence"
)

// Server represents the TCR game server
type Server struct {
	Host         string
	Port         int
	listener     net.Listener
	clients      map[string]*Client
	clientsMux   sync.Mutex
	gameConfig   *models.GameConfig
	configLoader *persistence.ConfigLoader
	basePath     string
	authManager  *AuthManager        // Add auth manager for user authentication
	matchmaker   *MatchmakingManager // Add matchmaking manager
}

// Client represents a connected client
type Client struct {
	ID       string
	Username string
	Conn     net.Conn
	Codec    *network.Codec
	GameID   string
	Server   *Server
}

// NewServer creates a new TCR server
func NewServer(host string, port int, basePath string) *Server {
	return &Server{
		Host:         host,
		Port:         port,
		clients:      make(map[string]*Client),
		basePath:     basePath,
		configLoader: persistence.NewConfigLoader(basePath),
		authManager:  NewAuthManager(basePath), // Initialize auth manager
	}
}

// Start starts the server and begins accepting connections
func (s *Server) Start() error {
	// Load game configuration
	var err error
	s.gameConfig, err = s.configLoader.LoadGameConfig()
	if err != nil {
		return fmt.Errorf("failed to load game configuration: %w", err)
	}

	// Initialize the matchmaking manager
	s.matchmaker = NewMatchmakingManager(s)

	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start server on %s: %w", addr, err)
	}

	log.Printf("Server started on %s", addr)

	// Accept connections in a goroutine
	go s.acceptConnections()

	return nil
}

// Stop stops the server and closes all connections
func (s *Server) Stop() error {
	if s.listener != nil {
		err := s.listener.Close()
		if err != nil {
			return fmt.Errorf("failed to close listener: %w", err)
		}

		// Close all client connections
		s.clientsMux.Lock()
		for _, client := range s.clients {
			client.Conn.Close()
		}
		s.clients = make(map[string]*Client)
		s.clientsMux.Unlock()
	}

	return nil
}

// acceptConnections accepts incoming connections and handles them
func (s *Server) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			if opErr, ok := err.(*net.OpError); ok && opErr.Temporary() {
				// If it's a temporary error, wait a bit and try again
				time.Sleep(time.Second)
				continue
			}
			// If the listener was closed, break out of the loop
			break
		}

		// Create a client for this connection
		client := &Client{
			ID:     generateID(), // You'd need to implement this function
			Conn:   conn,
			Codec:  network.NewCodec(conn),
			Server: s,
		}

		// Add to clients map
		s.clientsMux.Lock()
		s.clients[client.ID] = client
		s.clientsMux.Unlock()

		// Handle client in a goroutine
		go s.handleClient(client)
	}
}

// handleClient manages communication with a connected client
func (s *Server) handleClient(client *Client) {
	defer func() {
		// Remove client from clients map when disconnected
		s.clientsMux.Lock()
		delete(s.clients, client.ID)
		s.clientsMux.Unlock()

		// Close the connection
		client.Conn.Close()
		log.Printf("Client %s disconnected", client.ID)
	}()

	log.Printf("New client connected: %s from %s", client.ID, client.Conn.RemoteAddr())

	// Send a welcome message
	welcomePayload := &network.GameEventPayload{
		Message: "Welcome to Text Clash Royale! Please login with your username and password by 'login' command. Use 'help' for more commands.",
		Time:    time.Now(),
	}

	err := client.Codec.Send(network.MessageTypeGameEvent, welcomePayload)
	if err != nil {
		log.Printf("Error sending welcome message to client %s: %v", client.ID, err)
		return
	}

	// Main communication loop
	for {
		msg, err := client.Codec.Receive()
		if err != nil {
			log.Printf("Error receiving message from client %s: %v", client.ID, err)
			return
		}

		// Process the message
		if err := s.processMessage(client, msg); err != nil {
			log.Printf("Error processing message from client %s: %v", client.ID, err)

			// Send error message to client
			errorPayload := &network.ErrorPayload{
				Code:    500, // Internal server error
				Message: "Error processing your request",
			}
			client.Codec.Send(network.MessageTypeError, errorPayload)

			// For critical errors, disconnect the client
			if err.Error() == "critical error" {
				return
			}
		}
	}
}

// processMessage processes a message from a client
func (s *Server) processMessage(client *Client, msg *network.Message) error {
	switch msg.Type {
	case network.MessageTypeLogin:
		var loginPayload network.LoginPayload
		if err := network.ParsePayload(msg, &loginPayload); err != nil {
			return err
		}

		// Authenticate the user using our AuthManager
		playerData, err := s.authManager.AuthenticateUser(loginPayload.Username, loginPayload.Password)
		if err != nil {
			// Authentication failed
			authResultPayload := &network.AuthResultPayload{
				Success: false,
				Message: err.Error(),
			}
			return client.Codec.Send(network.MessageTypeAuthResult, authResultPayload)
		}

		// Authentication successful
		client.Username = playerData.Username

		// Register the user as active
		if err := s.authManager.RegisterActiveUser(playerData.Username, client.ID); err != nil {
			authResultPayload := &network.AuthResultPayload{
				Success: false,
				Message: err.Error(),
			}
			return client.Codec.Send(network.MessageTypeAuthResult, authResultPayload)
		}

		// Send successful authentication result
		authResultPayload := &network.AuthResultPayload{
			Success:  true,
			Message:  "Authentication successful",
			PlayerID: client.ID,
		}

		if err := client.Codec.Send(network.MessageTypeAuthResult, authResultPayload); err != nil {
			return err
		}

		// Send an info message about joining matchmaking
		infoPayload := &network.GameEventPayload{
			Message: "You can join the matchmaking queue by typing 'join'",
			Time:    time.Now(),
		}
		return client.Codec.Send(network.MessageTypeGameEvent, infoPayload)

	case network.MessageTypeJoinQueue:
		// Check if the client is authenticated
		if client.Username == "" {
			return client.Codec.Send(network.MessageTypeError, &network.ErrorPayload{
				Code:    401,
				Message: "You must be logged in to join matchmaking",
			})
		}

		// Add the client to the matchmaking queue
		s.matchmaker.AddToWaitingPool(client)
		return nil

	case network.MessageTypeQuit:
		// If the user has authenticated, unregister them
		if client.Username != "" {
			s.authManager.UnregisterActiveUser(client.Username)
			// Also remove them from the matchmaking queue
			s.matchmaker.RemoveFromWaitingPool(client.ID)
		}
		return fmt.Errorf("critical error") // This will cause the client to be disconnected

	default:
		// Handle messages from authenticated users
		if client.Username == "" {
			return client.Codec.Send(network.MessageTypeError, &network.ErrorPayload{
				Code:    401,
				Message: "You must be logged in first",
			})
		}

		// For Sprint 1, if it's a GameEvent type, we'll treat it as a chat message
		if msg.Type == network.MessageTypeGameEvent {
			var payload network.GameEventPayload
			if err := network.ParsePayload(msg, &payload); err != nil {
				return err
			}

			// Format the message with the username
			messagePayload := &network.GameEventPayload{
				Message: fmt.Sprintf("[%s]: %s", client.Username, payload.Message),
				Time:    time.Now(),
			}

			// Broadcast to all clients (simple chat implementation)
			s.clientsMux.Lock()
			for _, c := range s.clients {
				if c.Username != "" { // Only send to authenticated clients
					if err := c.Codec.Send(network.MessageTypeGameEvent, messagePayload); err != nil {
						log.Printf("Error sending message to client %s: %v", c.ID, err)
					}
				}
			}
			s.clientsMux.Unlock()

			return nil
		}

		// For other message types, just acknowledge receipt
		gameEventPayload := &network.GameEventPayload{
			Message: fmt.Sprintf("Received message of type %s", msg.Type),
			Time:    time.Now(),
		}
		return client.Codec.Send(network.MessageTypeGameEvent, gameEventPayload)
	}
}

// generateID generates a unique ID for a client
func generateID() string {
	return fmt.Sprintf("client-%d", time.Now().UnixNano())
}
