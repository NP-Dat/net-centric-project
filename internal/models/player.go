package models

// Player represents a player in the system with persistent data
type Player struct {
	ID             string `json:"id"`
	Username       string `json:"username"`
	HashedPassword string `json:"hashedPassword"`
	EXP            int    `json:"exp"`
	Level          int    `json:"level"`
}

// PlayerData represents the data structure for JSON persistence
type PlayerData struct {
	Username       string `json:"username"`
	HashedPassword string `json:"hashedPassword"`
	EXP            int    `json:"exp"`
	Level          int    `json:"level"`
}

// CalculateRequiredExp calculates the EXP required to reach the next level
// Level N+1 needs 100 * (1.1 ^ (N-1)) total EXP from the previous level
func CalculateRequiredExp(currentLevel int) int {
	if currentLevel < 1 {
		return 0
	}

	// Base EXP needed for Level 2 is 100
	baseExp := 100

	// For Level 1, return the base exp
	if currentLevel == 1 {
		return baseExp
	}

	// Calculate multiplier: 1.1^(currentLevel-1)
	multiplier := 1.0
	for i := 0; i < currentLevel-1; i++ {
		multiplier *= 1.1
	}

	return int(float64(baseExp) * multiplier)
}

// CalculateStatBoost returns the multiplier for stats based on the player's level
// Level N Stat = BaseStat * (1.1 ^ (N-1))
func CalculateStatBoost(level int) float64 {
	if level <= 1 {
		return 1.0
	}

	multiplier := 1.0
	for i := 0; i < level-1; i++ {
		multiplier *= 1.1
	}

	return multiplier
}
