// filepath: d:\Phuc Dat\IU\MY PROJECT\Golang\net-centric-project\internal\server\session.go
package server

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/NP-Dat/net-centric-project/internal/game"
	"github.com/NP-Dat/net-centric-project/internal/models"
	"github.com/NP-Dat/net-centric-project/internal/network"
	"github.com/NP-Dat/net-centric-project/internal/persistence"
)

// SessionManager handles game sessions for the TCR server
type SessionManager struct {
	server        *Server
	sessions      map[string]*GameSession
	sessionsMutex sync.RWMutex
	configLoader  *persistence.ConfigLoader
}

// GameSession represents a single game session between two players
type GameSession struct {
	ID           string
	Game         *game.Game
	Player1      *Client
	Player2      *Client
	GameMode     game.GameMode
	Active       bool
	LastActivity time.Time
}

// NewSessionManager creates a new session manager
func NewSessionManager(server *Server, configLoader *persistence.ConfigLoader) *SessionManager {
	return &SessionManager{
		server:       server,
		sessions:     make(map[string]*GameSession),
		configLoader: configLoader,
	}
}

// CreateSession creates a new game session between two players
func (sm *SessionManager) CreateSession(player1, player2 *Client, gameID string, gameMode game.GameMode) (*GameSession, error) {
	sm.sessionsMutex.Lock()
	defer sm.sessionsMutex.Unlock()

	// Check if session already exists
	if _, exists := sm.sessions[gameID]; exists {
		return nil, fmt.Errorf("game session with ID %s already exists", gameID)
	}

	// Load player data
	player1Data, err := sm.server.authManager.GetPlayerData(player1.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to load player1 data: %w", err)
	}

	player2Data, err := sm.server.authManager.GetPlayerData(player2.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to load player2 data: %w", err)
	}

	// Load game configuration needed for NewGame
	gameConfig, err := sm.configLoader.LoadGameConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load game configuration for session creation: %w", err)
	}

	// Convert config maps to maps of pointers as expected by game functions
	towerSpecsPtr := make(map[string]*models.TowerSpec)
	for id, spec := range gameConfig.Towers {
		specCopy := spec // Create a copy to avoid taking address of loop variable
		towerSpecsPtr[id] = &specCopy
	}
	troopSpecsPtr := make(map[string]*models.TroopSpec)
	for id, spec := range gameConfig.Troops {
		specCopy := spec // Create a copy
		troopSpecsPtr[id] = &specCopy
	}

	// Create new game instance, passing the maps of pointers
	gameInstance := game.NewGame(gameID, player1Data, player2Data, gameMode, towerSpecsPtr, troopSpecsPtr)

	// Create session struct
	session := &GameSession{
		ID:           gameID,
		Game:         gameInstance,
		Player1:      player1,
		Player2:      player2,
		GameMode:     gameMode,
		Active:       true,
		LastActivity: time.Now(),
	}

	// Initialize game state using the appropriate mode handler
	switch gameMode {
	case game.GameModeSimple:
		simpleHandler := game.NewSimpleModeHandler(gameInstance)
		// Pass the maps of pointers to StartGame as well
		if err := simpleHandler.StartGame(player1Data, player2Data, towerSpecsPtr, troopSpecsPtr); err != nil {
			return nil, fmt.Errorf("failed to start simple game mode: %w", err)
		}
		gameInstance.GameState = game.GameStateRunningSimple // Set state after successful start
	// case game.GameModeEnhanced: // Add this when Enhanced mode is implemented
	// 	enhancedHandler := game.NewEnhancedModeHandler(gameInstance)
	// 	if err := enhancedHandler.StartGame(player1Data, player2Data, gameConfig.Towers, gameConfig.Troops); err != nil {
	// 		return nil, fmt.Errorf("failed to start enhanced game mode: %w", err)
	// 	}
	// 	gameInstance.GameState = game.GameStateRunningEnhanced
	default:
		return nil, fmt.Errorf("unsupported game mode: %s", gameMode)
	}

	// Store session
	sm.sessions[gameID] = session

	// Set start time (already set within StartGame methods, but can confirm here)
	if gameInstance.StartTime.IsZero() {
		gameInstance.StartTime = time.Now()
	}

	// For enhanced mode, set the end time (should be handled by EnhancedModeHandler.StartGame)
	// if gameMode == game.GameModeEnhanced {
	// 	 gameInstance.EndTime = gameInstance.StartTime.Add(3 * time.Minute)
	// }

	// Associate clients with this game session
	player1.GameID = gameID
	player2.GameID = gameID

	// Start game session processing (e.g., sending initial state)
	go sm.runGameSession(session)

	return session, nil
}

// GetSession gets a game session by ID
func (sm *SessionManager) GetSession(gameID string) (*GameSession, bool) {
	sm.sessionsMutex.RLock()
	defer sm.sessionsMutex.RUnlock()

	session, exists := sm.sessions[gameID]
	return session, exists
}

// EndSession ends a game session
func (sm *SessionManager) EndSession(gameID string) {
	sm.sessionsMutex.Lock()
	defer sm.sessionsMutex.Unlock()

	session, exists := sm.sessions[gameID]
	if !exists {
		return
	}

	// Set session as inactive
	session.Active = false

	// Set game state as finished if not already
	if session.Game.GameState != game.GameStateFinished {
		session.Game.GameState = game.GameStateFinished
		if session.Game.EndTime.IsZero() {
			session.Game.EndTime = time.Now()
		}
	}

	// Remove session from active sessions
	delete(sm.sessions, gameID)

	// Clear game ID from clients
	if session.Player1 != nil {
		session.Player1.GameID = ""
	}
	if session.Player2 != nil {
		session.Player2.GameID = ""
	}

	// Notify players that the game has ended (already done in handleGameOver)
	log.Printf("Game session %s ended and cleaned up", gameID)
}

// runGameSession handles the main game loop for a session
func (sm *SessionManager) runGameSession(session *GameSession) {
	// For simple mode, just send initial game state to both players
	if session.GameMode == game.GameModeSimple {
		// Send initial game state to players
		sm.sendInitialGameState(session)
	} else if session.GameMode == game.GameModeEnhanced {
		// Enhanced mode not implemented in Sprint 1
		log.Printf("Enhanced mode not implemented in Sprint 1")
	}

	// For Sprint 1, we just initialize the game and don't manage the game loop yet
}

// sendInitialGameState sends the initial game state to both players
func (sm *SessionManager) sendInitialGameState(session *GameSession) {
	// Convert game state to network representation
	gameState := convertGameStateToPayload(session.Game, session.Player1.Username)

	// Prepare game start messages for players
	p1Payload := &network.GameStartPayload{
		GameID:           session.ID,
		OpponentUsername: session.Player2.Username,
		GameMode:         "simple",
		YourTurn:         true, // Player 1 goes first in simple mode
		InitialState:     gameState,
	}

	// For player 2, adjust the game state view
	p2GameState := convertGameStateToPayload(session.Game, session.Player2.Username)
	p2Payload := &network.GameStartPayload{
		GameID:           session.ID,
		OpponentUsername: session.Player1.Username,
		GameMode:         "simple",
		YourTurn:         false, // Player 2 goes second in simple mode
		InitialState:     p2GameState,
	}

	// Send game start messages
	err := session.Player1.Codec.Send(network.MessageTypeGameStart, p1Payload)
	if err != nil {
		log.Printf("Error sending game start to player %s: %v", session.Player1.Username, err)
	}

	err = session.Player2.Codec.Send(network.MessageTypeGameStart, p2Payload)
	if err != nil {
		log.Printf("Error sending game start to player %s: %v", session.Player2.Username, err)
	}

	// Log game start
	log.Printf("Game %s started between %s and %s", session.ID, session.Player1.Username, session.Player2.Username)

	// Send initial troop choices to Player 1 (who starts)
	if session.GameMode == game.GameModeSimple && session.Game.CurrentTurnPlayerIndex == 0 {
		sm.sendTroopChoicesToCurrentPlayer(session) // Player 1 is players[0]
	}
}

// sendTroopChoicesToCurrentPlayer generates and sends troop choices to the player whose turn it currently is.
func (sm *SessionManager) sendTroopChoicesToCurrentPlayer(session *GameSession) {
	if session.Game == nil || session.Game.GameState != game.GameStateRunningSimple {
		log.Printf("[Session %s] Cannot send troop choices: game not running or not simple mode.", session.ID)
		return
	}

	playerIndex := session.Game.CurrentTurnPlayerIndex
	currentPlayerInGame := session.Game.Players[playerIndex]
	if currentPlayerInGame == nil {
		log.Printf("[Session %s] Cannot send troop choices: current player at index %d is nil.", session.ID, playerIndex)
		return
	}

	// Get the game handler (assuming SimpleModeHandler for now)
	// This might need a more robust way to get the correct handler if multiple modes are active
	simpleHandler := game.NewSimpleModeHandler(session.Game) // Create a new handler instance for this operation

	troopChoicesPayload, err := simpleHandler.GenerateAndStoreTroopChoices(currentPlayerInGame)
	if err != nil {
		log.Printf("[Session %s] Error generating troop choices for player %s: %v", session.ID, currentPlayerInGame.Username, err)
		// Optionally, send an error message to the client if appropriate
		return
	}

	if troopChoicesPayload == nil || len(troopChoicesPayload.Choices) == 0 {
		log.Printf("[Session %s] No troop choices generated for player %s (perhaps no troops available).", session.ID, currentPlayerInGame.Username)
		// Send an empty choices message or a specific notification if desired
		// For now, we can send it. Client should handle empty choices gracefully.
	}

	// Determine which client (Player1 or Player2 of the session) is the current player
	var targetClient *Client
	if session.Player1.Username == currentPlayerInGame.Username { // Compare by a unique identifier like Username or ID
		targetClient = session.Player1
	} else if session.Player2.Username == currentPlayerInGame.Username {
		targetClient = session.Player2
	} else {
		log.Printf("[Session %s] Critical error: Could not match PlayerInGame %s to a session client.", session.ID, currentPlayerInGame.Username)
		return
	}

	if targetClient == nil || targetClient.Codec == nil {
		log.Printf("[Session %s] Cannot send troop choices to %s: client or codec is nil.", session.ID, currentPlayerInGame.Username)
		return
	}

	log.Printf("[Session %s] Sending troop choices to %s: %+v", session.ID, targetClient.Username, troopChoicesPayload.Choices)
	err = targetClient.Codec.Send(network.MessageTypeTroopChoices, troopChoicesPayload)
	if err != nil {
		log.Printf("[Session %s] Error sending troop choices to player %s: %v", session.ID, targetClient.Username, err)
	}
}

// convertGameStateToPayload converts a game.Game state to a network.GameStatePayload
func convertGameStateToPayload(game *game.Game, viewerUsername string) *network.GameStatePayload {
	payload := &network.GameStatePayload{
		Towers: make([]network.TowerInfo, 0, len(game.BoardState.Towers)),
		Troops: make([]network.TroopInfo, 0, len(game.BoardState.ActiveTroops)),
	}

	// Add towers to payload
	for _, tower := range game.BoardState.Towers {
		// Find owning player's username
		var ownerUsername string
		var ownerID string // Store owner ID for position check
		for _, player := range game.Players {
			if player.ID == tower.OwnerPlayerID {
				ownerUsername = player.Username
				ownerID = player.ID
				break
			}
		}

		// Determine position based on tower ID and SpecID
		position := "unknown"
		switch tower.SpecID {
		case "king_tower":
			position = "king"
		case "guard_tower":
			if tower.ID == fmt.Sprintf("guard1_%s", ownerID) {
				position = "guard1"
			} else if tower.ID == fmt.Sprintf("guard2_%s", ownerID) {
				position = "guard2"
			}
		}

		towerInfo := network.TowerInfo{
			ID:            tower.ID,
			SpecID:        tower.SpecID,
			Name:          tower.Name, // Keep original name like "Guard Tower 1"
			CurrentHP:     tower.CurrentHP,
			MaxHP:         tower.MaxHP,
			OwnerUsername: ownerUsername,
			Position:      position, // Use the determined position
		}
		payload.Towers = append(payload.Towers, towerInfo)
	}

	// Add troops to payload
	for _, troop := range game.BoardState.ActiveTroops {
		// Find owning player's username
		var ownerUsername string
		for _, player := range game.Players {
			if player.ID == troop.OwnerPlayerID {
				ownerUsername = player.Username
				break
			}
		}

		troopInfo := network.TroopInfo{
			InstanceID:    troop.InstanceID,
			SpecID:        troop.SpecID,
			Name:          troop.Name,
			CurrentHP:     troop.CurrentHP,
			MaxHP:         troop.MaxHP,
			OwnerUsername: ownerUsername,
			TargetTowerID: troop.TargetID,
		}
		payload.Troops = append(payload.Troops, troopInfo)
	}

	return payload
}

// HandleDeployTroop processes a troop deployment request from a client
func (sm *SessionManager) HandleDeployTroop(client *Client, troopID string) error {
	// Get the session for this client
	session, exists := sm.GetSession(client.GameID)
	if !exists {
		return fmt.Errorf("game session not found or not active for client %s", client.Username)
	}

	// Ensure the game is in the correct state (RunningSimple)
	if session.Game.GameState != game.GameStateRunningSimple {
		return fmt.Errorf("game %s is not in RunningSimple state", session.ID)
	}

	// Determine player index for the game logic
	playerIndex := -1
	var otherPlayerClient *Client // Re-declare otherPlayerClient
	if session.Player1.Username == client.Username {
		playerIndex = 0
		otherPlayerClient = session.Player2 // Assign otherPlayerClient
	} else if session.Player2.Username == client.Username {
		playerIndex = 1
		otherPlayerClient = session.Player1 // Assign otherPlayerClient
	} else {
		// This case was already handled by the playerIndex check, but defensive
		return fmt.Errorf("client %s not part of session %s for event broadcasting", client.Username, session.ID)
	}

	if playerIndex == -1 {
		return fmt.Errorf("client %s not found in game session %s", client.Username, session.ID)
	}

	// Check if it's the client's turn
	if session.Game.CurrentTurnPlayerIndex != playerIndex {
		errMsg := fmt.Sprintf("Not your turn. It is player %d's turn.", session.Game.CurrentTurnPlayerIndex)
		sendErr := client.Codec.Send(network.MessageTypeError, &network.ErrorPayload{Message: errMsg})
		if sendErr != nil {
			log.Printf("Error sending 'not your turn' error to %s: %v", client.Username, sendErr)
		}
		return fmt.Errorf(errMsg) // Also return error to stop further processing
	}

	// Get the appropriate game handler
	// For now, we assume SimpleModeHandler. This might need to be stored in GameSession or retrieved based on Game.Mode
	simpleHandler := game.NewSimpleModeHandler(session.Game)

	// Prepare action data
	actionData := map[string]interface{}{"troop_id": troopID}

	// Process the turn logic
	events, err := simpleHandler.ProcessTurn(playerIndex, "deploy_troop", actionData)
	if err != nil {
		log.Printf("Error processing turn for player %s in game %s: %v", client.Username, session.ID, err)
		// Send error to client
		sendErr := client.Codec.Send(network.MessageTypeError, &network.ErrorPayload{Message: err.Error()})
		if sendErr != nil {
			log.Printf("Error sending process turn error to %s: %v", client.Username, sendErr)
		}
		return err // Return the error from ProcessTurn
	}

	// Send game events that occurred during the turn (e.g., troop deployed, attacks, damage)
	for _, event := range events {
		// Send to current player (client)
		if err := client.Codec.Send(network.MessageTypeGameEvent, &event); err != nil {
			log.Printf("Error sending game event to %s: %v", client.Username, err)
		}
		// Send to the other player
		if otherPlayerClient != nil && otherPlayerClient.Codec != nil {
			if err := otherPlayerClient.Codec.Send(network.MessageTypeGameEvent, &event); err != nil {
				log.Printf("Error sending game event to %s: %v", otherPlayerClient.Username, err)
			}
		} else {
			log.Printf("Warning: otherPlayerClient or its codec is nil for game %s", session.ID)
		}
	}

	// After processing the turn, check for game over
	if session.Game.GameState == game.GameStateFinished {
		sm.handleGameOver(session)
		return nil // Game is over, no further turn actions
	}

	// If game is not over, update game state for all players
	sm.sendUpdatedGameState(session)

	// Notify players about the turn change
	sm.notifyTurnChange(session) // This will inform whose turn it is now

	// Send troop choices to the NEW current player
	sm.sendTroopChoicesToCurrentPlayer(session)

	return nil
}

// sendUpdatedGameState sends the current game state to both players
func (sm *SessionManager) sendUpdatedGameState(session *GameSession) {
	// Get game state for player 1's perspective
	p1GameState := convertGameStateToPayload(session.Game, session.Player1.Username)
	err := session.Player1.Codec.Send(network.MessageTypeStateUpdate, p1GameState)
	if err != nil {
		log.Printf("Error sending state update to player 1: %v", err)
	}

	// Get game state for player 2's perspective
	p2GameState := convertGameStateToPayload(session.Game, session.Player2.Username)
	err = session.Player2.Codec.Send(network.MessageTypeStateUpdate, p2GameState)
	if err != nil {
		log.Printf("Error sending state update to player 2: %v", err)
	}
}

// notifyTurnChange notifies players about whose turn it is
func (sm *SessionManager) notifyTurnChange(session *GameSession) {
	// Only applicable for Simple mode
	if session.GameMode != game.GameModeSimple {
		return
	}

	// Create turn change payloads for both players
	p1TurnChange := &network.TurnChangePayload{
		YourTurn: session.Game.CurrentTurnPlayerIndex == 0,
	}
	p2TurnChange := &network.TurnChangePayload{
		YourTurn: session.Game.CurrentTurnPlayerIndex == 1,
	}

	// Send turn change messages
	err := session.Player1.Codec.Send(network.MessageTypeTurnChange, p1TurnChange)
	if err != nil {
		log.Printf("Error sending turn change to player 1: %v", err)
	}

	err = session.Player2.Codec.Send(network.MessageTypeTurnChange, p2TurnChange)
	if err != nil {
		log.Printf("Error sending turn change to player 2: %v", err)
	}
}

// handleGameOver handles the end of a game
func (sm *SessionManager) handleGameOver(session *GameSession) {
	var winnerUsername string
	// var loserUsername string
	var reason string

	// Determine winner/loser from Game struct
	if session.Game.WinnerID != "" {
		if session.Game.WinnerID == session.Game.Players[0].ID { // Use Game.Players for ID
			winnerUsername = session.Player1.Username
			// loserUsername = session.Player2.Username
		} else {
			winnerUsername = session.Player2.Username
			// loserUsername = session.Player1.Username
		}
		reason = "King Tower destroyed"
	} else {
		// Handle draw or other conditions if needed (not applicable for Simple mode win)
		reason = "Game ended (unknown reason)" // Placeholder
	}

	// --- Calculate EXP --- (Simple Mode: Only EXP from destroyed towers)
	p1ExpEarned := calculateDestroyedTowerExp(session.Game.BoardState, session.Game.Players[1].ID, session.Game.TowerSpecs)
	p2ExpEarned := calculateDestroyedTowerExp(session.Game.BoardState, session.Game.Players[0].ID, session.Game.TowerSpecs)

	// --- Update Player Data ---
	// Load player data
	p1Data, err1 := persistence.LoadPlayerData(sm.server.basePath, session.Player1.Username)
	p2Data, err2 := persistence.LoadPlayerData(sm.server.basePath, session.Player2.Username)

	if err1 != nil {
		log.Printf("Error loading player data for %s: %v", session.Player1.Username, err1)
	} else {
		p1Data.EXP += p1ExpEarned
	}
	if err2 != nil {
		log.Printf("Error loading player data for %s: %v", session.Player2.Username, err2)
	} else {
		p2Data.EXP += p2ExpEarned
	}

	// Check for level ups and save data
	p1LeveledUp := false
	p2LeveledUp := false

	if p1Data != nil {
		for {
			requiredExp := models.CalculateRequiredExp(p1Data.Level)
			if p1Data.EXP >= requiredExp {
				p1Data.Level++
				p1Data.EXP -= requiredExp
				p1LeveledUp = true
			} else {
				break
			}
		}
		if err := persistence.SavePlayerData(sm.server.basePath, p1Data); err != nil {
			log.Printf("Error saving player data for %s: %v", p1Data.Username, err)
		}
	}

	if p2Data != nil {
		for {
			requiredExp := models.CalculateRequiredExp(p2Data.Level)
			if p2Data.EXP >= requiredExp {
				p2Data.Level++
				p2Data.EXP -= requiredExp
				p2LeveledUp = true
			} else {
				break
			}
		}
		if err := persistence.SavePlayerData(sm.server.basePath, p2Data); err != nil {
			log.Printf("Error saving player data for %s: %v", p2Data.Username, err)
		}
	}

	// Prepare game over payloads with updated data
	p1NewTotalExp := 0
	p1NewLevel := 0
	if p1Data != nil {
		p1NewTotalExp = p1Data.EXP
		p1NewLevel = p1Data.Level
	}
	p2NewTotalExp := 0
	p2NewLevel := 0
	if p2Data != nil {
		p2NewTotalExp = p2Data.EXP
		p2NewLevel = p2Data.Level
	}

	p1GameOver := &network.GameOverPayload{
		Winner:      winnerUsername,
		Reason:      reason,
		ExpEarned:   p1ExpEarned,
		NewTotalExp: p1NewTotalExp,
		NewLevel:    p1NewLevel,
		LeveledUp:   p1LeveledUp,
	}

	p2GameOver := &network.GameOverPayload{
		Winner:      winnerUsername,
		Reason:      reason,
		ExpEarned:   p2ExpEarned,
		NewTotalExp: p2NewTotalExp,
		NewLevel:    p2NewLevel,
		LeveledUp:   p2LeveledUp,
	}

	// Send game over messages
	if session.Player1 != nil && session.Player1.Codec != nil {
		err := session.Player1.Codec.Send(network.MessageTypeGameOver, p1GameOver)
		if err != nil {
			log.Printf("Error sending game over to player %s: %v", session.Player1.Username, err)
		}
	}

	if session.Player2 != nil && session.Player2.Codec != nil {
		err := session.Player2.Codec.Send(network.MessageTypeGameOver, p2GameOver)
		if err != nil {
			log.Printf("Error sending game over to player %s: %v", session.Player2.Username, err)
		}
	}

	// End the session (cleanup)
	sm.EndSession(session.ID)
}

// calculateDestroyedTowerExp calculates EXP earned from destroying opponent towers
func calculateDestroyedTowerExp(board *game.BoardState, opponentPlayerID string, towerSpecs map[string]*models.TowerSpec) int {
	expGained := 0
	for _, tower := range board.Towers {
		if tower.OwnerPlayerID == opponentPlayerID && tower.CurrentHP <= 0 {
			spec, exists := towerSpecs[tower.SpecID]
			if exists {
				expGained += spec.ExpYield
			}
		}
	}
	return expGained
}

// broadcastGameEvent sends a game event message to both players in a session.
// ... existing code ...
