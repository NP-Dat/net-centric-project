// filepath: d:\Phuc Dat\IU\MY PROJECT\Golang\net-centric-project\internal\game\logic_simple.go
package game

import (
	"fmt"
	"time"

	"github.com/NP-Dat/net-centric-project/internal/models"
	"github.com/NP-Dat/net-centric-project/internal/network"
	"github.com/google/uuid"
)

// SimpleModeHandler handles the logic for the Simple TCR mode
type SimpleModeHandler struct {
	game *Game
}

// NewSimpleModeHandler creates a new handler for Simple TCR mode
func NewSimpleModeHandler(game *Game) *SimpleModeHandler {
	return &SimpleModeHandler{
		game: game,
	}
}

// StartGame initializes the game state for Simple mode
func (h *SimpleModeHandler) StartGame(player1 *models.Player, player2 *models.Player, towerSpecs map[string]*models.TowerSpec, troopSpecs map[string]*models.TroopSpec) error {
	// Change game state to Running Simple
	h.game.GameState = GameStateRunningSimple
	h.game.StartTime = time.Now()

	// Initialize towers for both players based on specs and player levels
	err := h.initializeTowers(player1, player2, towerSpecs)
	if err != nil {
		return fmt.Errorf("failed to initialize towers: %w", err)
	}

	return nil
}

// initializeTowers creates the towers for both players
func (h *SimpleModeHandler) initializeTowers(player1 *models.Player, player2 *models.Player, towerSpecs map[string]*models.TowerSpec) error {
	// Create King and Guard towers for player 1
	if err := h.createPlayerTowers(h.game.Players[0], towerSpecs); err != nil {
		return err
	}

	// Create King and Guard towers for player 2
	if err := h.createPlayerTowers(h.game.Players[1], towerSpecs); err != nil {
		return err
	}

	return nil
}

// createPlayerTowers creates towers for a player based on tower specs
func (h *SimpleModeHandler) createPlayerTowers(player *PlayerInGame, towerSpecs map[string]*models.TowerSpec) error {
	// Get king tower spec
	kingSpec, exists := towerSpecs["king_tower"]
	if !exists {
		return fmt.Errorf("king tower spec not found")
	}

	// Get guard tower spec
	guardSpec, exists := towerSpecs["guard_tower"]
	if !exists {
		return fmt.Errorf("guard tower spec not found")
	}

	// Calculate level multiplier for player stats (10% increase per level)
	levelMultiplier := 1.0 + 0.1*float64(player.Level-1)

	// Create King Tower
	kingTower := &Tower{
		ID:            fmt.Sprintf("king_%s", player.ID),
		SpecID:        "king_tower",
		Name:          kingSpec.Name,
		CurrentHP:     int(float64(kingSpec.BaseHP) * levelMultiplier),
		MaxHP:         int(float64(kingSpec.BaseHP) * levelMultiplier),
		ATK:           int(float64(kingSpec.BaseATK) * levelMultiplier),
		DEF:           int(float64(kingSpec.BaseDEF) * levelMultiplier),
		CritChance:    kingSpec.CritChance,
		OwnerPlayerID: player.ID,
	}

	// Create Guard Tower 1
	guardTower1 := &Tower{
		ID:            fmt.Sprintf("guard1_%s", player.ID),
		SpecID:        "guard_tower",
		Name:          guardSpec.Name + " 1",
		CurrentHP:     int(float64(guardSpec.BaseHP) * levelMultiplier),
		MaxHP:         int(float64(guardSpec.BaseHP) * levelMultiplier),
		ATK:           int(float64(guardSpec.BaseATK) * levelMultiplier),
		DEF:           int(float64(guardSpec.BaseDEF) * levelMultiplier),
		CritChance:    guardSpec.CritChance,
		OwnerPlayerID: player.ID,
	}

	// Create Guard Tower 2
	guardTower2 := &Tower{
		ID:            fmt.Sprintf("guard2_%s", player.ID),
		SpecID:        "guard_tower",
		Name:          guardSpec.Name + " 2",
		CurrentHP:     int(float64(guardSpec.BaseHP) * levelMultiplier),
		MaxHP:         int(float64(guardSpec.BaseHP) * levelMultiplier),
		ATK:           int(float64(guardSpec.BaseATK) * levelMultiplier),
		DEF:           int(float64(guardSpec.BaseDEF) * levelMultiplier),
		CritChance:    guardSpec.CritChance,
		OwnerPlayerID: player.ID,
	}

	// Add towers to player's towers and the game board
	player.Towers[kingTower.ID] = kingTower
	player.Towers[guardTower1.ID] = guardTower1
	player.Towers[guardTower2.ID] = guardTower2

	h.game.BoardState.Towers[kingTower.ID] = kingTower
	h.game.BoardState.Towers[guardTower1.ID] = guardTower1
	h.game.BoardState.Towers[guardTower2.ID] = guardTower2

	return nil
}

// ProcessTurn processes a player's action during their turn
func (h *SimpleModeHandler) ProcessTurn(playerIndex int, action string, actionData map[string]interface{}) ([]network.GameEventPayload, error) {
	// Ensure it's the correct player's turn
	if h.game.CurrentTurnPlayerIndex != playerIndex {
		return nil, fmt.Errorf("not your turn")
	}

	var events []network.GameEventPayload

	// Get the current player
	currentPlayer := h.game.Players[playerIndex]

	// Get opponent player
	opponentIndex := (playerIndex + 1) % 2
	opponent := h.game.Players[opponentIndex]

	switch action {
	case "deploy_troop":
		// Deploy a troop
		troopID, ok := actionData["troop_id"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid troop ID")
		}

		// Deploy the troop (create an instance)
		troopEvents, err := h.deployTroop(currentPlayer, opponent, troopID)
		if err != nil {
			return nil, err
		}
		events = append(events, troopEvents...)

		// Process troop attacks (for already deployed troops)
		attackEvents, err := h.processTroopAttacks(currentPlayer, opponent)
		if err != nil {
			return nil, err
		}
		events = append(events, attackEvents...)

		// Process tower counterattacks
		counterEvents, err := h.processTowerCounterattacks(currentPlayer, opponent)
		if err != nil {
			return nil, err
		}
		events = append(events, counterEvents...)

		// Special handling for Queen troop (healing)
		if troopID == "queen" {
			healEvents, err := h.processQueenHealing(currentPlayer)
			if err != nil {
				return nil, err
			}
			events = append(events, healEvents...)
		}

	default:
		return nil, fmt.Errorf("invalid action: %s", action)
	}

	// Check if the game is over
	gameOverEvent := h.checkGameOver()
	if gameOverEvent != nil {
		events = append(events, *gameOverEvent)
		h.game.GameState = GameStateFinished
		h.game.EndTime = time.Now()
	} else {
		// End turn - switch to the next player
		h.nextTurn()

		// Add turn change event
		events = append(events, network.GameEventPayload{
			Message: fmt.Sprintf("Turn changed to player %s", h.game.Players[h.game.CurrentTurnPlayerIndex].Username),
			Time:    time.Now(),
		})
	}

	return events, nil
}

// deployTroop creates a new troop instance for the player
func (h *SimpleModeHandler) deployTroop(player *PlayerInGame, opponent *PlayerInGame, troopID string) ([]network.GameEventPayload, error) {
	// For Sprint 2, we'll implement basic deployment functionality
	var events []network.GameEventPayload

	// Add event for troop deployment
	events = append(events, network.GameEventPayload{
		Message: fmt.Sprintf("%s deployed a %s", player.Username, troopID),
		Time:    time.Now(),
	})

	// If it's the Queen troop, we handle it specially (heals instead of being deployed as an attacker)
	if troopID == "queen" {
		// Queen is handled in the processQueenHealing method
		return events, nil
	}

	// Generate a unique ID for the troop instance
	instanceID := uuid.New().String()

	// Create new troop instance (in a real implementation, you would look up stats from troopSpecs)
	// For Sprint 2, we'll just create a placeholder troop
	troop := &ActiveTroop{
		InstanceID:    instanceID,
		SpecID:        troopID,
		Name:          troopID, // In a full implementation, get the name from the troop spec
		CurrentHP:     100,     // Placeholder value, should be from troop spec
		MaxHP:         100,     // Placeholder value, should be from troop spec
		ATK:           50,      // Placeholder value, should be from troop spec
		DEF:           20,      // Placeholder value, should be from troop spec
		OwnerPlayerID: player.ID,
		DeployedTime:  time.Now(),
	}

	// Determine target tower based on Guard Tower 1 rule
	targetTower := h.findValidTarget(opponent)
	if targetTower != nil {
		troop.TargetID = targetTower.ID

		// Mark the tower as being targeted
		targetTower.TargetID = instanceID

		events = append(events, network.GameEventPayload{
			Message: fmt.Sprintf("%s's %s is targeting %s's %s", player.Username, troopID, opponent.Username, targetTower.Name),
			Time:    time.Now(),
		})
	}

	// Add troop to player's active troops and game board
	player.ActiveTroops[instanceID] = troop
	h.game.BoardState.ActiveTroops[instanceID] = troop

	return events, nil
}

// findValidTarget returns a valid tower target following the targeting rule:
// Guard Tower 1 must be destroyed before Guard Tower 2 or King Tower can be targeted
func (h *SimpleModeHandler) findValidTarget(opponent *PlayerInGame) *Tower {
	// Check if Guard Tower 1 exists and has HP > 0
	for id, tower := range opponent.Towers {
		if tower.Name == "Guard Tower 1" && tower.CurrentHP > 0 {
			return opponent.Towers[id]
		}
	}

	// If Guard Tower 1 is destroyed, check Guard Tower 2
	for id, tower := range opponent.Towers {
		if tower.Name == "Guard Tower 2" && tower.CurrentHP > 0 {
			return opponent.Towers[id]
		}
	}

	// If both Guard Towers are destroyed, target King Tower
	for id, tower := range opponent.Towers {
		if tower.Name == "King Tower" && tower.CurrentHP > 0 {
			return opponent.Towers[id]
		}
	}

	// No valid targets (should not happen in normal gameplay)
	return nil
}

// processTroopAttacks processes attacks from troops deployed in previous turns
func (h *SimpleModeHandler) processTroopAttacks(player *PlayerInGame, opponent *PlayerInGame) ([]network.GameEventPayload, error) {
	var events []network.GameEventPayload

	// Troops attack towers they're targeting
	for _, troop := range player.ActiveTroops {
		// Skip troops deployed this turn (they attack next turn)
		if time.Since(troop.DeployedTime) < time.Second*5 { // A reasonable threshold to identify "this turn" troops
			continue
		}

		// Find target tower
		targetTower, exists := h.game.BoardState.Towers[troop.TargetID]
		if !exists || targetTower.CurrentHP <= 0 {
			// Target is gone, find a new valid target
			newTarget := h.findValidTarget(opponent)
			if newTarget == nil {
				// No valid targets remain
				continue
			}
			troop.TargetID = newTarget.ID
			targetTower = newTarget
		}

		// Calculate damage (simplified for Sprint 2)
		damage := troop.ATK - targetTower.DEF
		if damage < 0 {
			damage = 0
		}

		// Apply damage
		targetTower.CurrentHP -= damage

		events = append(events, network.GameEventPayload{
			Message: fmt.Sprintf("%s's %s attacks %s's %s for %d damage",
				player.Username, troop.Name, opponent.Username, targetTower.Name, damage),
			Time: time.Now(),
		})

		// Check if tower was destroyed
		if targetTower.CurrentHP <= 0 {
			targetTower.CurrentHP = 0
			events = append(events, network.GameEventPayload{
				Message: fmt.Sprintf("%s's %s was destroyed!", opponent.Username, targetTower.Name),
				Time:    time.Now(),
			})
		}
	}

	return events, nil
}

// processTowerCounterattacks handles towers attacking troops that attacked them
func (h *SimpleModeHandler) processTowerCounterattacks(player *PlayerInGame, opponent *PlayerInGame) ([]network.GameEventPayload, error) {
	var events []network.GameEventPayload

	// Towers counterattack troops that attacked them
	for _, tower := range opponent.Towers {
		if tower.CurrentHP <= 0 || tower.TargetID == "" {
			continue
		}

		// Find the troop that attacked this tower
		targetTroop, exists := h.game.BoardState.ActiveTroops[tower.TargetID]
		if !exists {
			tower.TargetID = "" // Clear target if troop no longer exists
			continue
		}

		// Calculate damage
		damage := tower.ATK - targetTroop.DEF
		if damage < 0 {
			damage = 0
		}

		// Apply damage
		targetTroop.CurrentHP -= damage

		events = append(events, network.GameEventPayload{
			Message: fmt.Sprintf("%s's %s counterattacks %s's %s for %d damage",
				opponent.Username, tower.Name, player.Username, targetTroop.Name, damage),
			Time: time.Now(),
		})

		// Check if troop was defeated
		if targetTroop.CurrentHP <= 0 {
			targetTroop.CurrentHP = 0
			events = append(events, network.GameEventPayload{
				Message: fmt.Sprintf("%s's %s was defeated!", player.Username, targetTroop.Name),
				Time:    time.Now(),
			})

			// Remove defeated troop
			delete(player.ActiveTroops, targetTroop.InstanceID)
			delete(h.game.BoardState.ActiveTroops, targetTroop.InstanceID)

			// Clear tower's target
			tower.TargetID = ""
		}
	}

	return events, nil
}

// processQueenHealing handles the Queen troop's special healing ability
func (h *SimpleModeHandler) processQueenHealing(player *PlayerInGame) ([]network.GameEventPayload, error) {
	var events []network.GameEventPayload

	// Find tower with lowest HP percentage
	var lowestHPTower *Tower
	lowestHPPercentage := 100.0

	for _, tower := range player.Towers {
		if tower.CurrentHP <= 0 {
			continue // Skip destroyed towers
		}

		hpPercentage := float64(tower.CurrentHP) / float64(tower.MaxHP) * 100
		if lowestHPPercentage > hpPercentage {
			lowestHPPercentage = hpPercentage
			lowestHPTower = tower
		}
	}

	if lowestHPTower != nil {
		// Heal the tower by 300 HP, up to its maximum
		healAmount := 300
		beforeHP := lowestHPTower.CurrentHP
		lowestHPTower.CurrentHP += healAmount

		if lowestHPTower.CurrentHP > lowestHPTower.MaxHP {
			lowestHPTower.CurrentHP = lowestHPTower.MaxHP
		}

		actualHeal := lowestHPTower.CurrentHP - beforeHP

		events = append(events, network.GameEventPayload{
			Message: fmt.Sprintf("%s's Queen healed %s for %d HP",
				player.Username, lowestHPTower.Name, actualHeal),
			Time: time.Now(),
		})
	}

	return events, nil
}

// checkGameOver checks if the game is over (king tower destroyed)
func (h *SimpleModeHandler) checkGameOver() *network.GameEventPayload {
	// Check if either player's king tower is destroyed
	for playerIndex, player := range h.game.Players {
		for _, tower := range player.Towers {
			if tower.Name == "King Tower" && tower.CurrentHP <= 0 {
				// This player's king tower is destroyed, the other player wins
				winnerIndex := (playerIndex + 1) % 2
				return &network.GameEventPayload{
					Message: fmt.Sprintf("Game Over: %s wins! (King Tower destroyed)",
						h.game.Players[winnerIndex].Username),
					Time: time.Now(),
				}
			}
		}
	}

	return nil
}

// nextTurn advances to the next player's turn
func (h *SimpleModeHandler) nextTurn() {
	h.game.CurrentTurnPlayerIndex = (h.game.CurrentTurnPlayerIndex + 1) % 2
}

// GetGameState prepares a game state message to send to clients
func (h *SimpleModeHandler) GetGameState(playerIndex int) *network.GameStatePayload {
	gameState := &network.GameStatePayload{
		Towers: make([]network.TowerInfo, 0),
		Troops: make([]network.TroopInfo, 0),
	}

	// Add tower info
	for _, tower := range h.game.BoardState.Towers {
		playerUsername := h.game.Players[0].Username
		if tower.OwnerPlayerID == h.game.Players[1].ID {
			playerUsername = h.game.Players[1].Username
		}

		// Determine position based on tower name
		position := "unknown"
		if tower.Name == "King Tower" {
			position = "king"
		} else if tower.Name == "Guard Tower 1" {
			position = "guard1"
		} else if tower.Name == "Guard Tower 2" {
			position = "guard2"
		}

		gameState.Towers = append(gameState.Towers, network.TowerInfo{
			ID:            tower.ID,
			SpecID:        tower.SpecID,
			Name:          tower.Name,
			CurrentHP:     tower.CurrentHP,
			MaxHP:         tower.MaxHP,
			OwnerUsername: playerUsername,
			Position:      position,
		})
	}

	// Add troop info
	for _, troop := range h.game.BoardState.ActiveTroops {
		playerUsername := h.game.Players[0].Username
		if troop.OwnerPlayerID == h.game.Players[1].ID {
			playerUsername = h.game.Players[1].Username
		}

		gameState.Troops = append(gameState.Troops, network.TroopInfo{
			InstanceID:    troop.InstanceID,
			SpecID:        troop.SpecID,
			Name:          troop.Name,
			CurrentHP:     troop.CurrentHP,
			MaxHP:         troop.MaxHP,
			OwnerUsername: playerUsername,
			TargetTowerID: troop.TargetID,
		})
	}

	return gameState
}
