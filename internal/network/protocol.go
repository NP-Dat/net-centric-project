package network

import "time"

// MessageType defines the types of messages that can be exchanged
type MessageType string

// Define message types for client-server communication
const (
	// Client to Server message types
	MessageTypeLogin       MessageType = "login"
	MessageTypeDeployTroop MessageType = "deploy_troop"
	MessageTypeQuit        MessageType = "quit"

	// Server to Client message types
	MessageTypeAuthResult  MessageType = "auth_result"
	MessageTypeGameStart   MessageType = "game_start"
	MessageTypeStateUpdate MessageType = "state_update"
	MessageTypeGameEvent   MessageType = "game_event"
	MessageTypeGameOver    MessageType = "game_over"
	MessageTypeTurnChange  MessageType = "turn_change"
	MessageTypeError       MessageType = "error"
)

// Message is the base structure for all network messages
type Message struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
}

// ----- Client to Server Message Payloads -----

// LoginPayload represents the payload for a login message
type LoginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// DeployTroopPayload represents the payload for deploying a troop
type DeployTroopPayload struct {
	TroopID       string `json:"troop_id"`
	TargetTowerID string `json:"target_tower_id,omitempty"` // Optional, targeting may be implicit
}

// QuitPayload represents the payload for quitting a game
type QuitPayload struct {
	Reason string `json:"reason,omitempty"`
}

// ----- Server to Client Message Payloads -----

// AuthResultPayload represents the payload for authentication result
type AuthResultPayload struct {
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	PlayerID string `json:"player_id,omitempty"`
}

// GameStartPayload represents the payload when a game starts
type GameStartPayload struct {
	GameID           string            `json:"game_id"`
	OpponentUsername string            `json:"opponent_username"`
	GameMode         string            `json:"game_mode"` // "simple" or "enhanced"
	YourTurn         bool              `json:"your_turn"` // Only for Simple mode
	InitialState     *GameStatePayload `json:"initial_state"`
}

// TowerInfo contains information about a tower for state updates
type TowerInfo struct {
	ID            string `json:"id"`
	SpecID        string `json:"spec_id"`
	Name          string `json:"name"`
	CurrentHP     int    `json:"current_hp"`
	MaxHP         int    `json:"max_hp"`
	OwnerUsername string `json:"owner_username"`
	Position      string `json:"position"` // "king", "guard1", "guard2"
}

// TroopInfo contains information about a troop for state updates
type TroopInfo struct {
	InstanceID    string `json:"instance_id"`
	SpecID        string `json:"spec_id"`
	Name          string `json:"name"`
	CurrentHP     int    `json:"current_hp"`
	MaxHP         int    `json:"max_hp"`
	OwnerUsername string `json:"owner_username"`
	TargetTowerID string `json:"target_tower_id,omitempty"`
}

// GameStatePayload represents the current state of the game
type GameStatePayload struct {
	Towers       []TowerInfo `json:"towers"`
	Troops       []TroopInfo `json:"troops"`
	YourMana     int         `json:"your_mana,omitempty"`     // Only for Enhanced mode
	OpponentMana int         `json:"opponent_mana,omitempty"` // Only for Enhanced mode
	TimeLeft     int         `json:"time_left,omitempty"`     // Only for Enhanced mode, in seconds
}

// GameEventPayload represents a game event notification
type GameEventPayload struct {
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

// TurnChangePayload represents a turn change notification
type TurnChangePayload struct {
	YourTurn  bool      `json:"your_turn"`
	TimeoutAt time.Time `json:"timeout_at,omitempty"` // Optional, for turn time limits
}

// GameOverPayload represents the game over notification
type GameOverPayload struct {
	Winner      string `json:"winner,omitempty"` // Username of winner, empty if draw
	Reason      string `json:"reason"`           // e.g., "King tower destroyed", "Time expired"
	ExpEarned   int    `json:"exp_earned"`
	NewTotalExp int    `json:"new_total_exp"`
	NewLevel    int    `json:"new_level"`
	LeveledUp   bool   `json:"leveled_up"`
}

// ErrorPayload represents an error message
type ErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
