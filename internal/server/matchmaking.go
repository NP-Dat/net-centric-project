package server

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/NP-Dat/net-centric-project/internal/game"
	"github.com/NP-Dat/net-centric-project/internal/network"
)

// MatchmakingManager handles matchmaking functionality
type MatchmakingManager struct {
	server      *Server
	waitingPool []*Client // Pool of clients waiting for a match
	poolMutex   sync.Mutex
	gameCounter int // Counter for generating game IDs
}

// NewMatchmakingManager creates a new matchmaking manager
func NewMatchmakingManager(server *Server) *MatchmakingManager {
	mm := &MatchmakingManager{
		server:      server,
		waitingPool: make([]*Client, 0),
	}

	// Start the matchmaking process in a separate goroutine
	go mm.matchmakingLoop()

	return mm
}

// AddToWaitingPool adds a client to the waiting pool for matchmaking
func (mm *MatchmakingManager) AddToWaitingPool(client *Client) {
	mm.poolMutex.Lock()
	defer mm.poolMutex.Unlock()

	// Check if client is already in the pool
	for _, c := range mm.waitingPool {
		if c.ID == client.ID {
			return // Client already in pool
		}
	}

	// Add client to the pool
	mm.waitingPool = append(mm.waitingPool, client)

	// Inform the client they've been added to the matchmaking queue
	event := &network.GameEventPayload{
		Message: "You have been added to the matchmaking queue. Waiting for opponent...",
		Time:    time.Now(),
	}

	if err := client.Codec.Send(network.MessageTypeGameEvent, event); err != nil {
		log.Printf("Error sending matchmaking event to client %s: %v", client.ID, err)
	}
}

// RemoveFromWaitingPool removes a client from the waiting pool
func (mm *MatchmakingManager) RemoveFromWaitingPool(clientID string) {
	mm.poolMutex.Lock()
	defer mm.poolMutex.Unlock()

	// Find and remove the client from the pool
	for i, client := range mm.waitingPool {
		if client.ID == clientID {
			mm.waitingPool = append(mm.waitingPool[:i], mm.waitingPool[i+1:]...)
			break
		}
	}
}

// matchmakingLoop continuously checks for possible matches
func (mm *MatchmakingManager) matchmakingLoop() {
	ticker := time.NewTicker(1 * time.Second) // Check every second
	defer ticker.Stop()

	for range ticker.C {
		if err := mm.tryMatchmaking(); err != nil {
			log.Printf("Error during matchmaking: %v", err)
		}
	}
}

// tryMatchmaking attempts to create matches between waiting clients
func (mm *MatchmakingManager) tryMatchmaking() error {
	mm.poolMutex.Lock()
	defer mm.poolMutex.Unlock()

	// For Sprint 1, simply match the first two waiting clients
	if len(mm.waitingPool) >= 2 {
		player1 := mm.waitingPool[0]
		player2 := mm.waitingPool[1]

		// Remove the matched players from the pool
		mm.waitingPool = mm.waitingPool[2:]

		// Create a new game for these players
		mm.gameCounter++
		gameID := fmt.Sprintf("game-%d", mm.gameCounter)

		// Start game using session manager
		mm.startGame(player1, player2, gameID)
	}

	return nil
}

// startGame initiates a new game between two players
func (mm *MatchmakingManager) startGame(player1, player2 *Client, gameID string) {
	log.Printf("Starting game %s between %s and %s", gameID, player1.Username, player2.Username)

	// Use the session manager to create and start the game
	session, err := mm.server.sessionManager.CreateSession(player1, player2, gameID, game.GameModeSimple)
	if err != nil {
		log.Printf("Error creating game session: %v", err)

		// Notify players about the error
		errorMsg := &network.GameEventPayload{
			Message: fmt.Sprintf("Failed to start game: %v", err),
			Time:    time.Now(),
		}

		player1.Codec.Send(network.MessageTypeGameEvent, errorMsg)
		player2.Codec.Send(network.MessageTypeGameEvent, errorMsg)
		return
	}

	log.Printf("Game session %s created successfully. Mode: %v", session.ID, session.GameMode)
}
