package models

// GameConfig contains maps/slices of TowerSpec and TroopSpec
type GameConfig struct {
	Towers map[string]TowerSpec `json:"towers"`
	Troops map[string]TroopSpec `json:"troops"`
}

// TowerSpec defines the base specifications for a tower type
type TowerSpec struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	BaseHP     int     `json:"baseHP"`
	BaseATK    int     `json:"baseATK"`
	BaseDEF    int     `json:"baseDEF"`
	CritChance float64 `json:"critChance"` // Percentage (e.g., 5.0 for 5%)
	ExpYield   int     `json:"expYield"`   // EXP gained when this tower is destroyed
}

// TroopSpec defines the base specifications for a troop type
type TroopSpec struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	BaseHP     int    `json:"baseHP"`
	BaseATK    int    `json:"baseATK"`
	BaseDEF    int    `json:"baseDEF"`
	ManaCost   int    `json:"manaCost"`
	ExpYield   int    `json:"expYield"` // EXP gained when this troop is deployed/destroyed
	Special    string `json:"special"`  // Description of any special ability
	HasSpecial bool   `json:"hasSpecial"`
}
