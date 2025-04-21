package client

import (
	"fmt"
	"net"
	"sync"

	"github.com/NP-Dat/net-centric-project/internal/network"
)

// Client represents the TCR game client
type Client struct {
	Host            string
	Port            int
	Username        string
	conn            net.Conn
	codec           *network.Codec
	messageHandlers map[network.MessageType]MessageHandler
	handlersMutex   sync.RWMutex
	connected       bool
	disconnectChan  chan struct{}
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
		return fmt.Errorf("already connected to server")
	}

	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)
	var err error
	c.conn, err = net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to server at %s: %w", addr, err)
	}

	c.codec = network.NewCodec(c.conn)
	c.connected = true

	// Start receiving messages in a goroutine
	go c.receiveMessages()

	return nil
}

// Disconnect disconnects from the server
func (c *Client) Disconnect() error {
	if !c.connected {
		return nil
	}

	// Try to send a quit message
	if c.codec != nil {
		_ = c.codec.Send(network.MessageTypeQuit, &network.QuitPayload{
			Reason: "Client disconnecting",
		})
	}

	// Close the connection
	err := c.conn.Close()
	c.connected = false
	close(c.disconnectChan)

	return err
}

// RegisterHandler registers a handler for a specific message type
func (c *Client) RegisterHandler(msgType network.MessageType, handler MessageHandler) {
	c.handlersMutex.Lock()
	defer c.handlersMutex.Unlock()
	c.messageHandlers[msgType] = handler
}

// RemoveHandler removes a handler for a specific message type
func (c *Client) RemoveHandler(msgType network.MessageType) {
	c.handlersMutex.Lock()
	defer c.handlersMutex.Unlock()
	delete(c.messageHandlers, msgType)
}

// Send sends a message to the server
func (c *Client) Send(msgType network.MessageType, payload interface{}) error {
	if !c.connected {
		return fmt.Errorf("not connected to server")
	}

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
		return fmt.Errorf("not connected to server")
	}

	c.Username = username

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
		return fmt.Errorf("not connected to server")
	}

	if c.Username == "" {
		return fmt.Errorf("must be logged in to join matchmaking")
	}

	// Send join queue message - empty payload is fine for Sprint 1
	return c.Send(network.MessageTypeJoinQueue, map[string]interface{}{})
}

// receiveMessages continuously receives and processes messages from the server
func (c *Client) receiveMessages() {
	defer func() {
		c.connected = false

		// Notify about disconnection if not already notified
		select {
		case <-c.disconnectChan:
			// Already closed
		default:
			close(c.disconnectChan)
		}
	}()

	for {
		msg, err := c.codec.Receive()
		if err != nil {
			// Connection closed or error
			return
		}

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
			fmt.Printf("Error handling message of type %s: %v\n", msg.Type, err)
		}
	} else {
		// Default handler for unhandled message types
		fmt.Printf("Received message of type %s with no handler\n", msg.Type)

		// If it's an error message, print it
		if msg.Type == network.MessageTypeError {
			var errorPayload network.ErrorPayload
			if err := network.ParsePayload(msg, &errorPayload); err == nil {
				fmt.Printf("Error from server: [%d] %s\n", errorPayload.Code, errorPayload.Message)
			}
		}
	}
}
