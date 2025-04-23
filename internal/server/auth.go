package server

import (
	"errors"
	"log"
	"sync"

	"github.com/NP-Dat/net-centric-project/internal/models"
	"github.com/NP-Dat/net-centric-project/internal/persistence"
	"golang.org/x/crypto/bcrypt"
)

// AuthManager handles authentication-related functionality
type AuthManager struct {
	basePath    string
	activeUsers map[string]string // Maps usernames to client IDs
	usersMutex  sync.RWMutex
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(basePath string) *AuthManager {
	return &AuthManager{
		basePath:    basePath,
		activeUsers: make(map[string]string),
	}
}

// GetPlayerData retrieves player data by username
func (am *AuthManager) GetPlayerData(username string) (*models.Player, error) {
	// Try to load player data from the persistence layer
	playerData, err := persistence.LoadPlayerData(am.basePath, username)
	if err != nil {
		log.Printf("Error loading player data for %s: %v", username, err)
		return nil, errors.New("error loading player data")
	}

	if playerData == nil {
		return nil, errors.New("player not found")
	}

	// Convert PlayerData to Player
	player := &models.Player{
		ID:             username, // Use username as ID for simplicity
		Username:       playerData.Username,
		HashedPassword: playerData.HashedPassword,
		EXP:            playerData.EXP,
		Level:          playerData.Level,
	}

	return player, nil
}

// AuthenticateUser authenticates a user with the given username and password
func (am *AuthManager) AuthenticateUser(username, password string) (*models.PlayerData, error) {
	// Validation: Check if the username or password is empty
	if username == "" || password == "" {
		return nil, errors.New("username and password cannot be empty")
	}

	// Try to load the player data from the persistence layer
	playerData, err := persistence.LoadPlayerData(am.basePath, username)
	if err != nil {
		log.Printf("Error loading player data for %s: %v", username, err)
		return nil, errors.New("error loading player data")
	}

	// If the player doesn't exist, create a new one with the given credentials
	if playerData == nil {
		// Hash the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, errors.New("error creating user account")
		}

		// Create a new player data object
		playerData = &models.PlayerData{
			Username:       username,
			HashedPassword: string(hashedPassword),
			EXP:            0,
			Level:          1, // Starting at level 1
		}

		// Save the new player data
		if err := persistence.SavePlayerData(am.basePath, playerData); err != nil {
			log.Printf("Error saving new player data for %s: %v", username, err)
			return nil, errors.New("error creating user account")
		}

		log.Printf("Created new account for user: %s", username)
	} else {
		// Player exists, verify the password
		err = bcrypt.CompareHashAndPassword([]byte(playerData.HashedPassword), []byte(password))
		if err != nil {
			return nil, errors.New("invalid username or password")
		}
	}

	return playerData, nil
}

// RegisterActiveUser registers a user as active with their client ID
func (am *AuthManager) RegisterActiveUser(username string, clientID string) error {
	am.usersMutex.Lock()
	defer am.usersMutex.Unlock()

	// Check if the user is already logged in
	if existingClientID, exists := am.activeUsers[username]; exists {
		if existingClientID != clientID {
			return errors.New("user already logged in from another client")
		}
		// If the same client ID, they're already registered
		return nil
	}

	am.activeUsers[username] = clientID
	log.Printf("User %s registered as active with client ID %s", username, clientID)
	return nil
}

// UnregisterActiveUser removes a user from the active users list
func (am *AuthManager) UnregisterActiveUser(username string) {
	am.usersMutex.Lock()
	defer am.usersMutex.Unlock()
	delete(am.activeUsers, username)
	log.Printf("User %s unregistered from active users", username)
}

// GetActiveUserCount returns the number of currently active users
func (am *AuthManager) GetActiveUserCount() int {
	am.usersMutex.RLock()
	defer am.usersMutex.RUnlock()
	return len(am.activeUsers)
}

// IsUserActive checks if a user is currently active
func (am *AuthManager) IsUserActive(username string) bool {
	am.usersMutex.RLock()
	defer am.usersMutex.RUnlock()
	_, exists := am.activeUsers[username]
	return exists
}
