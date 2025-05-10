package client

import (
	"fmt"
	"net"
	"sync"

	"github.com/NP-Dat/net-centric-project/internal/network"
	"github.com/NP-Dat/net-centric-project/pkg/logger"
)

// Client represents the TCR game client
type Client struct {
	Host                string
	Port                int
	Username            string
	conn                net.Conn
	codec               *network.Codec
	messageHandlers     map[network.MessageType]MessageHandler
	handlersMutex       sync.RWMutex
	connected           bool
	disconnectChan      chan struct{}
	currentTroopChoices []network.TroopChoiceInfo // Stores the troop choices received from the server
}

// MessageHandler is a function that handles a specific type of message
type MessageHandler func(msg *network.Message) error

// NewClient creates a new TCR client
func NewClient(host string, port int) *Client {
	return &Client{
		Host:            host,
		Port:            port,
		messageHandlers: make(map[network.MessageType]MessageHandler),
		disconnectChan:  make(chan struct{}),
	}
}

// Connect connects to the server
func (c *Client) Connect() error {
	// Check if already connected
	if c.connected {
		logger.Client.Warn("Attempted to connect while already connected to server")
		return fmt.Errorf("already connected to server")
	}

	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)
	var err error
	c.conn, err = net.Dial("tcp", addr)
	if err != nil {
		logger.Client.Error("Failed to connect to server at %s: %v", addr, err)
		return fmt.Errorf("failed to connect to server at %s: %w", addr, err)
	}

	c.codec = network.NewCodec(c.conn)
	c.connected = true

	logger.Client.Info("Connected to server at %s", addr)

	// Start receiving messages in a goroutine
	go c.receiveMessages()

	return nil
}

// Disconnect disconnects from the server
func (c *Client) Disconnect() error {
	if !c.connected {
		logger.Client.Debug("Disconnect called but client is not connected")
		return nil
	}

	// Try to send a quit message
	if c.codec != nil {
		logger.Client.Info("Sending quit message to server")
		_ = c.codec.Send(network.MessageTypeQuit, &network.QuitPayload{
			Reason: "Client disconnecting",
		})
	}

	// Close the connection
	err := c.conn.Close()
	if err != nil {
		logger.Client.Error("Error closing connection: %v", err)
	} else {
		logger.Client.Info("Connection closed successfully")
	}

	c.connected = false
	close(c.disconnectChan)

	return err
}

// RegisterHandler registers a handler for a specific message type
func (c *Client) RegisterHandler(msgType network.MessageType, handler MessageHandler) {
	c.handlersMutex.Lock()
	defer c.handlersMutex.Unlock()
	c.messageHandlers[msgType] = handler
	logger.Client.Debug("Registered handler for message type: %s", msgType)
}

// RemoveHandler removes a handler for a specific message type
func (c *Client) RemoveHandler(msgType network.MessageType) {
	c.handlersMutex.Lock()
	defer c.handlersMutex.Unlock()
	delete(c.messageHandlers, msgType)
	logger.Client.Debug("Removed handler for message type: %s", msgType)
}

// Send sends a message to the server
func (c *Client) Send(msgType network.MessageType, payload interface{}) error {
	if !c.connected {
		logger.Client.Error("Attempted to send message when not connected")
		return fmt.Errorf("not connected to server")
	}

	logger.Client.Debug("Sending message of type %s to server", msgType)
	return c.codec.Send(msgType, payload)
}

// IsConnected returns whether the client is connected to the server
func (c *Client) IsConnected() bool {
	return c.connected
}

// WaitForDisconnect blocks until the client is disconnected
func (c *Client) WaitForDisconnect() {
	<-c.disconnectChan
}

// LoginWithCredentials attempts to log in with the provided username and password
func (c *Client) LoginWithCredentials(username, password string) error {
	if !c.connected {
		logger.Client.Error("Attempted to login when not connected")
		return fmt.Errorf("not connected to server")
	}

	c.Username = username
	logger.Client.Info("Attempting to login with username: %s", username)

	// Send login message
	loginPayload := &network.LoginPayload{
		Username: username,
		Password: password,
	}

	return c.Send(network.MessageTypeLogin, loginPayload)
}

// JoinMatchmaking sends a request to join the matchmaking queue
func (c *Client) JoinMatchmaking() error {
	if !c.connected {
		logger.Client.Error("Attempted to join matchmaking when not connected")
		return fmt.Errorf("not connected to server")
	}

	if c.Username == "" {
		logger.Client.Warn("Attempted to join matchmaking while not logged in")
		return fmt.Errorf("must be logged in to join matchmaking")
	}

	logger.Client.Info("Requesting to join matchmaking queue")
	// Send join queue message - empty payload is fine for Sprint 1
	return c.Send(network.MessageTypeJoinQueue, map[string]interface{}{})
}

// receiveMessages continuously receives and processes messages from the server
func (c *Client) receiveMessages() {
	defer func() {
		// Handle panics
		if r := recover(); r != nil {
			logger.Client.Error("Panic in receiveMessages: %v", r)
		}

		c.connected = false

		// Notify about disconnection if not already notified
		select {
		case <-c.disconnectChan:
			// Already closed
		default:
			close(c.disconnectChan)
		}

		logger.Client.Info("Stopped receiving messages from server")
	}()

	logger.Client.Debug("Started message receiving loop")

	for {
		msg, err := c.codec.Receive()
		if err != nil {
			// Connection closed or error
			logger.Client.Error("Error receiving message from server: %v", err)
			return
		}

		logger.Client.Debug("Received message of type %s from server", msg.Type)

		// Process the received message
		c.processMessage(msg)
	}
}

// processMessage processes a message received from the server
func (c *Client) processMessage(msg *network.Message) {
	c.handlersMutex.RLock()
	handler, exists := c.messageHandlers[msg.Type]
	c.handlersMutex.RUnlock()

	if exists {
		if err := handler(msg); err != nil {
			logger.Client.Error("Error handling message of type %s: %v", msg.Type, err)
		} else {
			logger.Client.Debug("Successfully handled message of type %s", msg.Type)
		}
	} else {
		// Default handler for unhandled message types
		logger.Client.Warn("Received message of type %s with no registered handler", msg.Type)

		// If it's an error message, print it
		if msg.Type == network.MessageTypeError {
			var errorPayload network.ErrorPayload
			if err := network.ParsePayload(msg, &errorPayload); err == nil {
				logger.Client.Error("Error from server: [%d] %s", errorPayload.Code, errorPayload.Message)
			} else {
				logger.Client.Error("Received error message with invalid payload: %v", err)
			}
		}
	}
}

// SetCurrentTroopChoices sets the available troop choices for the client for the current turn.
func (c *Client) SetCurrentTroopChoices(choices []network.TroopChoiceInfo) {
	c.handlersMutex.Lock() // Reuse handlersMutex for thread safety, or add a new one if contention is an issue
	defer c.handlersMutex.Unlock()
	c.currentTroopChoices = choices
}

// GetCurrentTroopChoices retrieves the troop choices. It returns a copy to prevent external modification.
func (c *Client) GetCurrentTroopChoices() []network.TroopChoiceInfo {
	c.handlersMutex.RLock()
	defer c.handlersMutex.RUnlock()
	if c.currentTroopChoices == nil {
		return nil
	}
	// Return a copy to ensure the internal slice isn't modified by callers
	choicesCopy := make([]network.TroopChoiceInfo, len(c.currentTroopChoices))
	copy(choicesCopy, c.currentTroopChoices)
	return choicesCopy
}
