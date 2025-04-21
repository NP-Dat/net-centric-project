package network

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
)

// Codec provides functions for encoding and decoding messages over TCP
type Codec struct {
	conn   net.Conn
	reader *bufio.Reader
}

// NewCodec creates a new codec for the given connection
func NewCodec(conn net.Conn) *Codec {
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
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Add newline as message delimiter
	data = append(data, '\n')

	_, err = c.conn.Write(data)
	if err != nil {
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
			return nil, err
		}
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	var msg Message
	err = json.Unmarshal(data, &msg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &msg, nil
}

// ParsePayload parses the raw payload into the specified type
func ParsePayload(msg *Message, target interface{}) error {
	// Re-encode the payload to JSON
	data, err := json.Marshal(msg.Payload)
	if err != nil {
		return fmt.Errorf("failed to re-encode payload: %w", err)
	}

	// Decode into the target type
	err = json.Unmarshal(data, target)
	if err != nil {
		return fmt.Errorf("failed to decode payload: %w", err)
	}

	return nil
}
