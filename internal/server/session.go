// filepath: d:\Phuc Dat\IU\MY PROJECT\Golang\net-centric-project\internal\server\session.go
package server

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/NP-Dat/net-centric-project/internal/game"
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

	// Create new game
	gameInstance := game.NewGame(gameID, player1Data, player2Data, gameMode)

	// Create session
	session := &GameSession{
		ID:           gameID,
		Game:         gameInstance,
		Player1:      player1,
		Player2:      player2,
		GameMode:     gameMode,
		Active:       true,
		LastActivity: time.Now(),
	}

	// Initialize game state from config
	if err := sm.initializeGameState(session); err != nil {
		return nil, fmt.Errorf("failed to initialize game state: %w", err)
	}

	// Store session
	sm.sessions[gameID] = session

	// Set appropriate status based on game mode
	if gameMode == game.GameModeSimple {
		gameInstance.GameState = game.GameStateRunningSimple
	} else {
		gameInstance.GameState = game.GameStateRunningEnhanced
	}

	// Set start time
	gameInstance.StartTime = time.Now()

	// For enhanced mode, set the end time
	if gameMode == game.GameModeEnhanced {
		gameInstance.EndTime = gameInstance.StartTime.Add(3 * time.Minute)
	}

	// Associate clients with this game session
	player1.GameID = gameID
	player2.GameID = gameID

	// Start game session in a goroutine
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

	// Set game state as finished
	session.Game.GameState = game.GameStateFinished

	// Remove session from active sessions
	delete(sm.sessions, gameID)

	// Notify players that the game has ended
	log.Printf("Game session %s ended", gameID)
}

// initializeGameState sets up the initial game state with towers for each player
func (sm *SessionManager) initializeGameState(session *GameSession) error {
	// Load tower configurations
	gameConfig, err := sm.configLoader.LoadGameConfig()
	if err != nil {
		return fmt.Errorf("failed to load game configuration: %w", err)
	}

	// Create towers for player 1
	kingTowerSpec := gameConfig.Towers["king_tower"]
	guardTowerSpec := gameConfig.Towers["guard_tower"]

	// Player 1 King Tower
	kingTower1 := &game.Tower{
		ID:            fmt.Sprintf("p1_king_%s", session.ID),
		SpecID:        "king_tower",
		Name:          "King Tower",
		CurrentHP:     applyLevelMultiplier(kingTowerSpec.BaseHP, session.Game.Players[0].Level),
		MaxHP:         applyLevelMultiplier(kingTowerSpec.BaseHP, session.Game.Players[0].Level),
		ATK:           applyLevelMultiplier(kingTowerSpec.BaseATK, session.Game.Players[0].Level),
		DEF:           applyLevelMultiplier(kingTowerSpec.BaseDEF, session.Game.Players[0].Level),
		CritChance:    kingTowerSpec.CritChance,
		OwnerPlayerID: session.Game.Players[0].ID,
	}

	// Player 1 Guard Tower 1
	guardTower1_1 := &game.Tower{
		ID:            fmt.Sprintf("p1_guard1_%s", session.ID),
		SpecID:        "guard_tower",
		Name:          "Guard Tower 1",
		CurrentHP:     applyLevelMultiplier(guardTowerSpec.BaseHP, session.Game.Players[0].Level),
		MaxHP:         applyLevelMultiplier(guardTowerSpec.BaseHP, session.Game.Players[0].Level),
		ATK:           applyLevelMultiplier(guardTowerSpec.BaseATK, session.Game.Players[0].Level),
		DEF:           applyLevelMultiplier(guardTowerSpec.BaseDEF, session.Game.Players[0].Level),
		CritChance:    guardTowerSpec.CritChance,
		OwnerPlayerID: session.Game.Players[0].ID,
	}

	// Player 1 Guard Tower 2
	guardTower1_2 := &game.Tower{
		ID:            fmt.Sprintf("p1_guard2_%s", session.ID),
		SpecID:        "guard_tower",
		Name:          "Guard Tower 2",
		CurrentHP:     applyLevelMultiplier(guardTowerSpec.BaseHP, session.Game.Players[0].Level),
		MaxHP:         applyLevelMultiplier(guardTowerSpec.BaseHP, session.Game.Players[0].Level),
		ATK:           applyLevelMultiplier(guardTowerSpec.BaseATK, session.Game.Players[0].Level),
		DEF:           applyLevelMultiplier(guardTowerSpec.BaseDEF, session.Game.Players[0].Level),
		CritChance:    guardTowerSpec.CritChance,
		OwnerPlayerID: session.Game.Players[0].ID,
	}

	// Player 2 King Tower
	kingTower2 := &game.Tower{
		ID:            fmt.Sprintf("p2_king_%s", session.ID),
		SpecID:        "king_tower",
		Name:          "King Tower",
		CurrentHP:     applyLevelMultiplier(kingTowerSpec.BaseHP, session.Game.Players[1].Level),
		MaxHP:         applyLevelMultiplier(kingTowerSpec.BaseHP, session.Game.Players[1].Level),
		ATK:           applyLevelMultiplier(kingTowerSpec.BaseATK, session.Game.Players[1].Level),
		DEF:           applyLevelMultiplier(kingTowerSpec.BaseDEF, session.Game.Players[1].Level),
		CritChance:    kingTowerSpec.CritChance,
		OwnerPlayerID: session.Game.Players[1].ID,
	}

	// Player 2 Guard Tower 1
	guardTower2_1 := &game.Tower{
		ID:            fmt.Sprintf("p2_guard1_%s", session.ID),
		SpecID:        "guard_tower",
		Name:          "Guard Tower 1",
		CurrentHP:     applyLevelMultiplier(guardTowerSpec.BaseHP, session.Game.Players[1].Level),
		MaxHP:         applyLevelMultiplier(guardTowerSpec.BaseHP, session.Game.Players[1].Level),
		ATK:           applyLevelMultiplier(guardTowerSpec.BaseATK, session.Game.Players[1].Level),
		DEF:           applyLevelMultiplier(guardTowerSpec.BaseDEF, session.Game.Players[1].Level),
		CritChance:    guardTowerSpec.CritChance,
		OwnerPlayerID: session.Game.Players[1].ID,
	}

	// Player 2 Guard Tower 2
	guardTower2_2 := &game.Tower{
		ID:            fmt.Sprintf("p2_guard2_%s", session.ID),
		SpecID:        "guard_tower",
		Name:          "Guard Tower 2",
		CurrentHP:     applyLevelMultiplier(guardTowerSpec.BaseHP, session.Game.Players[1].Level),
		MaxHP:         applyLevelMultiplier(guardTowerSpec.BaseHP, session.Game.Players[1].Level),
		ATK:           applyLevelMultiplier(guardTowerSpec.BaseATK, session.Game.Players[1].Level),
		DEF:           applyLevelMultiplier(guardTowerSpec.BaseDEF, session.Game.Players[1].Level),
		CritChance:    guardTowerSpec.CritChance,
		OwnerPlayerID: session.Game.Players[1].ID,
	}

	// Add towers to board state
	session.Game.BoardState.Towers[kingTower1.ID] = kingTower1
	session.Game.BoardState.Towers[guardTower1_1.ID] = guardTower1_1
	session.Game.BoardState.Towers[guardTower1_2.ID] = guardTower1_2
	session.Game.BoardState.Towers[kingTower2.ID] = kingTower2
	session.Game.BoardState.Towers[guardTower2_1.ID] = guardTower2_1
	session.Game.BoardState.Towers[guardTower2_2.ID] = guardTower2_2

	// Add towers to player's tower list
	session.Game.Players[0].Towers[kingTower1.ID] = kingTower1
	session.Game.Players[0].Towers[guardTower1_1.ID] = guardTower1_1
	session.Game.Players[0].Towers[guardTower1_2.ID] = guardTower1_2
	session.Game.Players[1].Towers[kingTower2.ID] = kingTower2
	session.Game.Players[1].Towers[guardTower2_1.ID] = guardTower2_1
	session.Game.Players[1].Towers[guardTower2_2.ID] = guardTower2_2

	return nil
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
}

// convertGameStateToPayload converts a game.Game state to a network.GameStatePayload
func convertGameStateToPayload(game *game.Game, viewerUsername string) *network.GameStatePayload {
	payload := &network.GameStatePayload{
		Towers: make([]network.TowerInfo, 0, len(game.BoardState.Towers)),
		Troops: make([]network.TroopInfo, 0, len(game.BoardState.ActiveTroops)),
	}

	// Add towers to payload
	for _, tower := range game.BoardState.Towers {
		// Figure out the position
		position := "king"
		if tower.SpecID == "guard_tower" {
			if tower.ID[9:10] == "1" { // Assuming ID format like p1_guard1_...
				position = "guard1"
			} else {
				position = "guard2"
			}
		}

		// Find owning player's username
		var ownerUsername string
		for _, player := range game.Players {
			if player.ID == tower.OwnerPlayerID {
				ownerUsername = player.Username
				break
			}
		}

		towerInfo := network.TowerInfo{
			ID:            tower.ID,
			SpecID:        tower.SpecID,
			Name:          tower.Name,
			CurrentHP:     tower.CurrentHP,
			MaxHP:         tower.MaxHP,
			OwnerUsername: ownerUsername,
			Position:      position,
		}
		payload.Towers = append(payload.Towers, towerInfo)
	}

	// Add troops to payload (none in initial state for Sprint 1)
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

// applyLevelMultiplier applies the level multiplier to a base stat
// Each level increases stats by 10% cumulatively
func applyLevelMultiplier(baseStat int, level int) int {
	if level <= 1 {
		return baseStat
	}
	// Calculate 1.1^(level-1)
	multiplier := 1.0
	for i := 0; i < level-1; i++ {
		multiplier *= 1.1
	}
	return int(float64(baseStat) * multiplier)
}

// HandleDeployTroop processes a troop deployment request from a client
func (sm *SessionManager) HandleDeployTroop(client *Client, troopID string) error {
	// Get the session for this client
	session, exists := sm.GetSession(client.GameID)
	if !exists {
		return fmt.Errorf("game session not found")
	}

	// Check if the game is in Simple mode
	if session.GameMode != game.GameModeSimple {
		return client.Codec.Send(network.MessageTypeError, &network.ErrorPayload{
			Code:    400,
			Message: "Deploy command only supported in Simple mode for Sprint 2",
		})
	}

	// Verify the game is active
	if !session.Active || session.Game.GameState != game.GameStateRunningSimple {
		return client.Codec.Send(network.MessageTypeError, &network.ErrorPayload{
			Code:    400,
			Message: "Game is not active",
		})
	}

	// Determine which player is making the request
	var playerIndex int
	var otherPlayerClient *Client

	if client.Username == session.Player1.Username {
		playerIndex = 0
		otherPlayerClient = session.Player2
	} else if client.Username == session.Player2.Username {
		playerIndex = 1
		otherPlayerClient = session.Player1
	} else {
		return fmt.Errorf("client is not a player in this game")
	}

	// Verify it's the player's turn
	if session.Game.CurrentTurnPlayerIndex != playerIndex {
		return client.Codec.Send(network.MessageTypeError, &network.ErrorPayload{
			Code:    400,
			Message: "It's not your turn",
		})
	}

	// Validate the troop ID (can be enhanced later to check against available troop specs)
	validTroops := []string{"pawn", "bishop", "rook", "knight", "prince", "queen"}
	isValid := false
	for _, valid := range validTroops {
		if troopID == valid {
			isValid = true
			break
		}
	}

	if !isValid {
		return client.Codec.Send(network.MessageTypeError, &network.ErrorPayload{
			Code:    400,
			Message: "Invalid troop ID. Valid options are: pawn, bishop, rook, knight, prince, queen",
		})
	}

	// Create a SimpleModeHandler if needed
	var simpleModeHandler *game.SimpleModeHandler
	if session.Game.GameState == game.GameStateRunningSimple {
		simpleModeHandler = game.NewSimpleModeHandler(session.Game)
	}

	// Process the turn
	events, err := simpleModeHandler.ProcessTurn(playerIndex, "deploy_troop", map[string]interface{}{
		"troop_id": troopID,
	})
	if err != nil {
		return client.Codec.Send(network.MessageTypeError, &network.ErrorPayload{
			Code:    500,
			Message: "Error processing turn: " + err.Error(),
		})
	}

	// Send game events to both players
	for _, event := range events {
		// Send to current player
		err := client.Codec.Send(network.MessageTypeGameEvent, &event)
		if err != nil {
			log.Printf("Error sending game event to %s: %v", client.Username, err)
		}

		// Send to the other player
		err = otherPlayerClient.Codec.Send(network.MessageTypeGameEvent, &event)
		if err != nil {
			log.Printf("Error sending game event to %s: %v", otherPlayerClient.Username, err)
		}
	}

	// Send updated game state to both players
	sm.sendUpdatedGameState(session)

	// If the game is over, handle the end of game
	if session.Game.GameState == game.GameStateFinished {
		sm.handleGameOver(session)
	} else {
		// If not over, notify about turn change
		sm.notifyTurnChange(session)
	}

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
	// In Sprint 2, just set the basic game over message
	var winner string
	var reason string

	// Determine winner by checking king towers
	for playerIndex, player := range session.Game.Players {
		for _, tower := range player.Towers {
			if tower.Name == "King Tower" && tower.CurrentHP <= 0 {
				// This player lost because their king tower was destroyed
				winnerIndex := (playerIndex + 1) % 2
				if winnerIndex == 0 {
					winner = session.Player1.Username
				} else {
					winner = session.Player2.Username
				}
				reason = "King Tower destroyed"
				break
			}
		}
	}

	// Create game over payloads
	var p1Exp int
	if winner == session.Player1.Username {
		p1Exp = 200
	} else {
		p1Exp = 0
	}

	var p2Exp int
	if winner == session.Player2.Username {
		p2Exp = 200
	} else {
		p2Exp = 0
	}

	p1GameOver := &network.GameOverPayload{
		Winner:      winner,
		Reason:      reason,
		ExpEarned:   p1Exp,
		NewTotalExp: 0,
		NewLevel:    0,
		LeveledUp:   false,
	}

	p2GameOver := &network.GameOverPayload{
		Winner:      winner,
		Reason:      reason,
		ExpEarned:   p2Exp,
		NewTotalExp: 0,
		NewLevel:    0,
		LeveledUp:   false,
	}

	// Send game over messages
	err := session.Player1.Codec.Send(network.MessageTypeGameOver, p1GameOver)
	if err != nil {
		log.Printf("Error sending game over to player 1: %v", err)
	}

	err = session.Player2.Codec.Send(network.MessageTypeGameOver, p2GameOver)
	if err != nil {
		log.Printf("Error sending game over to player 2: %v", err)
	}

	// End the session
	sm.EndSession(session.ID)
}
