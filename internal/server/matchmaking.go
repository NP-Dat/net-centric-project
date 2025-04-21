package server

import (
	"fmt"
	"log"
	"sync"
	"time"

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

		// For Sprint 1, just notify both players that they've been matched
		mm.startGame(player1, player2, gameID)
	}

	return nil
}

// startGame initiates a new game between two players
func (mm *MatchmakingManager) startGame(player1, player2 *Client, gameID string) {
	log.Printf("Starting game %s between %s and %s", gameID, player1.Username, player2.Username)

	// For Sprint 1, we'll just notify players that they've been matched
	// In future sprints, we'd set up the actual game state

	// Set the game ID for both clients
	player1.GameID = gameID
	player2.GameID = gameID

	// Send game start event to player 1
	gameStartPayload1 := &network.GameStartPayload{
		GameID:           gameID,
		OpponentUsername: player2.Username,
		GameMode:         "simple",
		YourTurn:         true,                        // Player 1 goes first
		InitialState:     &network.GameStatePayload{}, // Empty for Sprint 1
	}

	// Send game start event to player 2
	gameStartPayload2 := &network.GameStartPayload{
		GameID:           gameID,
		OpponentUsername: player1.Username,
		GameMode:         "simple",
		YourTurn:         false,                       // Player 2 goes second
		InitialState:     &network.GameStatePayload{}, // Empty for Sprint 1
	}

	// Send the start game messages
	if err := player1.Codec.Send(network.MessageTypeGameStart, gameStartPayload1); err != nil {
		log.Printf("Error sending game start to player1 %s: %v", player1.ID, err)
	}

	if err := player2.Codec.Send(network.MessageTypeGameStart, gameStartPayload2); err != nil {
		log.Printf("Error sending game start to player2 %s: %v", player2.ID, err)
	}
}
