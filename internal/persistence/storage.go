package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/NP-Dat/net-centric-project/internal/models"
)

// ConfigLoader is responsible for loading game configuration from JSON files
type ConfigLoader struct {
	BasePath string
}

// NewConfigLoader creates a new ConfigLoader with the given base path
func NewConfigLoader(basePath string) *ConfigLoader {
	return &ConfigLoader{
		BasePath: basePath,
	}
}

// LoadTowerSpecs loads tower specifications from the towers.json file
func (c *ConfigLoader) LoadTowerSpecs() (map[string]models.TowerSpec, error) {
	filePath := filepath.Join(c.BasePath, "config", "towers.json")

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read towers config file: %w", err)
	}

	// Parse the JSON
	var config struct {
		Towers map[string]models.TowerSpec `json:"towers"`
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse towers config file: %w", err)
	}

	return config.Towers, nil
}

// LoadTroopSpecs loads troop specifications from the troops.json file
func (c *ConfigLoader) LoadTroopSpecs() (map[string]models.TroopSpec, error) {
	filePath := filepath.Join(c.BasePath, "config", "troops.json")

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read troops config file: %w", err)
	}

	// Parse the JSON
	var config struct {
		Troops map[string]models.TroopSpec `json:"troops"`
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse troops config file: %w", err)
	}

	return config.Troops, nil
}

// LoadGameConfig loads both tower and troop specifications and returns a GameConfig
func (c *ConfigLoader) LoadGameConfig() (*models.GameConfig, error) {
	towers, err := c.LoadTowerSpecs()
	if err != nil {
		return nil, err
	}

	troops, err := c.LoadTroopSpecs()
	if err != nil {
		return nil, err
	}

	return &models.GameConfig{
		Towers: towers,
		Troops: troops,
	}, nil
}

// SavePlayerData saves player data to a JSON file in the players directory
func SavePlayerData(basePath string, playerData *models.PlayerData) error {
	// Create the directory if it doesn't exist
	dirPath := filepath.Join(basePath, "data", "players")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create players directory: %w", err)
	}

	filePath := filepath.Join(dirPath, fmt.Sprintf("%s.json", playerData.Username))

	// Marshal the player data to JSON
	data, err := json.MarshalIndent(playerData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode player data: %w", err)
	}

	// Write to file
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to save player data file: %w", err)
	}

	return nil
}

// LoadPlayerData loads a player's data from their JSON file
func LoadPlayerData(basePath string, username string) (*models.PlayerData, error) {
	filePath := filepath.Join(basePath, "data", "players", fmt.Sprintf("%s.json", username))

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		// If the file doesn't exist, it's not necessarily an error - could be a new player
		if os.IsNotExist(err) {
			return nil, nil // Return nil without error to indicate player doesn't exist
		}
		return nil, fmt.Errorf("failed to read player data file: %w", err)
	}

	// Parse the JSON
	var playerData models.PlayerData
	err = json.Unmarshal(data, &playerData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse player data file: %w", err)
	}

	return &playerData, nil
}
