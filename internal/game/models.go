package game

import (
	"sync"
	"time"

	"github.com/NP-Dat/net-centric-project/internal/models"
)

// GameState represents the current state of a game
type GameState string

const (
	GameStateWaiting         GameState = "WAITING"
	GameStateRunningSimple   GameState = "RUNNING_SIMPLE"
	GameStateRunningEnhanced GameState = "RUNNING_ENHANCED"
	GameStateFinished        GameState = "FINISHED"
)

// GameMode represents the mode of the game
type GameMode string

const (
	GameModeSimple   GameMode = "SIMPLE"
	GameModeEnhanced GameMode = "ENHANCED"
)

// Game represents a game session between two players
type Game struct {
	ID                     string
	Players                [2]*PlayerInGame
	GameState              GameState
	GameMode               GameMode
	CurrentTurnPlayerIndex int         // Index of the player whose turn it is (Simple mode)
	StartTime              time.Time   // Game start time
	EndTime                time.Time   // Game end time or expected end time
	BoardState             *BoardState // Current state of the game board
	Mutex                  sync.Mutex  // For thread safety
}

// PlayerInGame represents a player within the context of a game
type PlayerInGame struct {
	ID           string
	Username     string
	Level        int
	GameID       string
	CurrentMana  int // For Enhanced mode
	Towers       map[string]*Tower
	ActiveTroops map[string]*ActiveTroop
	PlayerIndex  int         // 0 or 1
	Connection   interface{} // Will be replaced with net.Conn in implementation
}

// Tower represents an instance of a tower in the game
type Tower struct {
	ID            string
	SpecID        string // Reference to the tower spec
	Name          string
	CurrentHP     int
	MaxHP         int
	ATK           int
	DEF           int
	CritChance    float64
	OwnerPlayerID string
	TargetID      string // ID of the troop currently targeting this tower
}

// ActiveTroop represents a troop instance that has been deployed in the game
type ActiveTroop struct {
	InstanceID    string
	SpecID        string // Reference to the troop spec
	Name          string
	CurrentHP     int
	MaxHP         int
	ATK           int
	DEF           int
	OwnerPlayerID string
	TargetID      string // ID of the tower this troop is targeting
	DeployedTime  time.Time
}

// BoardState represents the current state of the game board
type BoardState struct {
	Towers       map[string]*Tower
	ActiveTroops map[string]*ActiveTroop
}

// NewGame creates a new game between two players
func NewGame(id string, player1 *models.Player, player2 *models.Player, mode GameMode) *Game {
	game := &Game{
		ID:        id,
		GameState: GameStateWaiting,
		GameMode:  mode,
		StartTime: time.Time{},
		EndTime:   time.Time{},
		BoardState: &BoardState{
			Towers:       make(map[string]*Tower),
			ActiveTroops: make(map[string]*ActiveTroop),
		},
		CurrentTurnPlayerIndex: 0, // Player 1 starts in Simple mode
	}

	// Initialize players in game
	game.Players[0] = &PlayerInGame{
		ID:           player1.ID,
		Username:     player1.Username,
		Level:        player1.Level,
		GameID:       id,
		CurrentMana:  5, // Starting mana in Enhanced mode
		Towers:       make(map[string]*Tower),
		ActiveTroops: make(map[string]*ActiveTroop),
		PlayerIndex:  0,
	}

	game.Players[1] = &PlayerInGame{
		ID:           player2.ID,
		Username:     player2.Username,
		Level:        player2.Level,
		GameID:       id,
		CurrentMana:  5, // Starting mana in Enhanced mode
		Towers:       make(map[string]*Tower),
		ActiveTroops: make(map[string]*ActiveTroop),
		PlayerIndex:  1,
	}

	return game
}
