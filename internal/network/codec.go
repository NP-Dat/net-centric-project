package network

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"github.com/NP-Dat/net-centric-project/pkg/logger"
)

// Codec provides functions for encoding and decoding messages over TCP
type Codec struct {
	conn   net.Conn
	reader *bufio.Reader
}

// NewCodec creates a new codec for the given connection
func NewCodec(conn net.Conn) *Codec {
	logger.Network.Debug("Creating new codec for connection from %s", conn.RemoteAddr())
	return &Codec{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}
}

// Send encodes a Message and sends it over the connection
func (c *Codec) Send(msgType MessageType, payload interface{}) error {
	msg := Message{
		Type:    msgType,
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Network.Error("Failed to marshal message of type %s: %v", msgType, err)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Add newline as message delimiter
	data = append(data, '\n')

	logger.Network.Debug("Sending message of type %s to %s (data size: %d bytes)",
		msgType, c.conn.RemoteAddr(), len(data))

	_, err = c.conn.Write(data)
	if err != nil {
		logger.Network.Error("Failed to send message to %s: %v", c.conn.RemoteAddr(), err)
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// Receive reads a Message from the connection and decodes it
func (c *Codec) Receive() (*Message, error) {
	// Read until newline
	data, err := c.reader.ReadBytes('\n')
	if err != nil {
		if err == io.EOF {
			logger.Network.Debug("Connection closed by peer %s", c.conn.RemoteAddr())
			return nil, err
		}
		logger.Network.Error("Failed to read message from %s: %v", c.conn.RemoteAddr(), err)
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	logger.Network.Debug("Received %d bytes from %s", len(data), c.conn.RemoteAddr())

	var msg Message
	err = json.Unmarshal(data, &msg)
	if err != nil {
		logger.Network.Error("Failed to unmarshal message from %s: %v", c.conn.RemoteAddr(), err)
		// Log the data that failed to unmarshal (limited to first 100 bytes for safety)
		if len(data) > 100 {
			logger.Network.Debug("Invalid message data (first 100 bytes): %s", string(data[:100]))
		} else {
			logger.Network.Debug("Invalid message data: %s", string(data))
		}
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	logger.Network.Debug("Successfully decoded message of type %s from %s", msg.Type, c.conn.RemoteAddr())
	return &msg, nil
}

// ParsePayload parses the raw payload into the specified type
func ParsePayload(msg *Message, target interface{}) error {
	if msg == nil {
		logger.Network.Error("Cannot parse payload: message is nil")
		return fmt.Errorf("cannot parse nil message")
	}

	// Re-encode the payload to JSON
	data, err := json.Marshal(msg.Payload)
	if err != nil {
		logger.Network.Error("Failed to re-encode payload for message type %s: %v", msg.Type, err)
		return fmt.Errorf("failed to re-encode payload: %w", err)
	}

	// Decode into the target type
	err = json.Unmarshal(data, target)
	if err != nil {
		logger.Network.Error("Failed to decode payload for message type %s: %v", msg.Type, err)
		// Log the payload that failed to unmarshal (limited to first 100 bytes for safety)
		if len(data) > 100 {
			logger.Network.Debug("Invalid payload data (first 100 bytes): %s", string(data[:100]))
		} else {
			logger.Network.Debug("Invalid payload data: %s", string(data))
		}
		return fmt.Errorf("failed to decode payload: %w", err)
	}

	logger.Network.Debug("Successfully parsed payload for message type %s", msg.Type)
	return nil
}
