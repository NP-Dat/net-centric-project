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
	h.game.Mutex.Lock() // Lock for thread safety
	defer h.game.Mutex.Unlock()

	// Ensure it's the correct player's turn
	if h.game.CurrentTurnPlayerIndex != playerIndex {
		return nil, fmt.Errorf("not your turn")
	}
	if h.game.GameState != GameStateRunningSimple {
		return nil, fmt.Errorf("game is not running in simple mode")
	}

	var events []network.GameEventPayload

	// Get the current player and opponent
	currentPlayer := h.game.Players[playerIndex]
	opponentIndex := (playerIndex + 1) % 2
	opponent := h.game.Players[opponentIndex]

	// --- Turn Sequence ---
	// 1. Player Action (e.g., Deploy Troop)
	switch action {
	case "deploy_troop":
		troopID, ok := actionData["troop_id"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid troop ID in action data")
		}

		// Deploy the troop (create an instance) or handle Queen heal
		deployEvents, err := h.deployTroop(currentPlayer, opponent, troopID)
		if err != nil {
			return nil, fmt.Errorf("failed to deploy troop: %w", err)
		}
		events = append(events, deployEvents...)

		// Special handling for Queen troop (healing) - happens immediately on deploy turn
		if troopID == "queen" {
			healEvents, err := h.processQueenHealing(currentPlayer)
			if err != nil {
				// Log error but maybe continue turn? Or return error?
				fmt.Printf("Error processing queen healing: %v\n", err) // Log error
				// Decide if this should halt the turn
			}
			events = append(events, healEvents...)
		}

	default:
		return nil, fmt.Errorf("invalid action for Simple Mode: %s", action)
	}

	// 2. Process attacks from *existing* troops (deployed in previous turns)
	attackEvents, err := h.processTroopAttacks(currentPlayer, opponent)
	if err != nil {
		return nil, fmt.Errorf("error during troop attacks: %w", err)
	}
	events = append(events, attackEvents...)

	// Check for game over after player's troop attacks
	gameOver, winnerIdx, loserIdx := h.checkGameOver()
	if gameOver {
		h.game.GameState = GameStateFinished
		h.game.EndTime = time.Now()
		h.game.WinnerID = h.game.Players[winnerIdx].ID
		h.game.LoserID = h.game.Players[loserIdx].ID
		events = append(events, network.GameEventPayload{
			Message: fmt.Sprintf("Game Over: %s wins! (King Tower destroyed)", h.game.Players[winnerIdx].Username),
			Time:    time.Now(),
		})
		// TODO: Award EXP based on destroyed towers (handled in session manager)
		return events, nil // Game over, return immediately
	}

	// 3. Process tower counterattacks (opponent's towers attack player's troops)
	// Towers counterattack troops that attacked them *this turn*.
	counterEvents, err := h.processTowerCounterattacks(currentPlayer, opponent)
	if err != nil {
		return nil, fmt.Errorf("error during tower counterattacks: %w", err)
	}
	events = append(events, counterEvents...)

	// 4. Cleanup defeated units (troops defeated by counterattacks)
	h.cleanupDefeatedUnits(currentPlayer, opponent) // Cleanup again after counterattacks

	// 5. Check for game over again (e.g., if a counterattack defeated the last troop needed for some condition, though unlikely in Simple mode)
	// This check might be redundant here for simple mode win condition (King Tower destruction)
	// but good practice for more complex scenarios.
	gameOver, winnerIdx, loserIdx = h.checkGameOver()
	if gameOver {
		h.game.GameState = GameStateFinished
		h.game.EndTime = time.Now()
		h.game.WinnerID = h.game.Players[winnerIdx].ID
		h.game.LoserID = h.game.Players[loserIdx].ID
		events = append(events, network.GameEventPayload{
			Message: fmt.Sprintf("Game Over: %s wins! (King Tower destroyed)", h.game.Players[winnerIdx].Username),
			Time:    time.Now(),
		})
		// TODO: Award EXP based on destroyed towers (handled in session manager)
		return events, nil // Game over
	}

	// 6. End turn - switch to the next player
	h.nextTurn()

	// Add turn change event
	events = append(events, network.GameEventPayload{
		Message: fmt.Sprintf("Turn changed to player %s", h.game.Players[h.game.CurrentTurnPlayerIndex].Username),
		Time:    time.Now(),
	})

	return events, nil
}

// deployTroop creates a new troop instance for the player
func (h *SimpleModeHandler) deployTroop(player *PlayerInGame, opponent *PlayerInGame, troopID string) ([]network.GameEventPayload, error) {
	var events []network.GameEventPayload

	// Look up troop spec from game config (assuming game has access to it)
	// This part needs access to the loaded troop specifications.
	// Let's assume h.game has a TroopSpecs map[string]*models.TroopSpec
	troopSpec, exists := h.game.TroopSpecs[troopID] // Need to ensure TroopSpecs is populated in the Game struct
	if !exists {
		return nil, fmt.Errorf("troop spec not found for ID: %s", troopID)
	}

	// Add event for troop deployment
	events = append(events, network.GameEventPayload{
		Message: fmt.Sprintf("%s deployed %s", player.Username, troopSpec.Name),
		Time:    time.Now(),
	})

	// If it's the Queen troop, we handle it specially (heals instead of being deployed as an attacker)
	if troopID == "queen" {
		// Queen is handled in the processQueenHealing method, called later in ProcessTurn
		return events, nil
	}

	// Generate a unique ID for the troop instance
	instanceID := uuid.New().String()

	// Calculate level multiplier for player stats (10% increase per level)
	levelMultiplier := 1.0 + 0.1*float64(player.Level-1)

	// Create new troop instance using stats from spec and player level
	troop := &ActiveTroop{
		InstanceID:    instanceID,
		SpecID:        troopID,
		Name:          troopSpec.Name,
		CurrentHP:     int(float64(troopSpec.BaseHP) * levelMultiplier),
		MaxHP:         int(float64(troopSpec.BaseHP) * levelMultiplier),
		ATK:           int(float64(troopSpec.BaseATK) * levelMultiplier),
		DEF:           int(float64(troopSpec.BaseDEF) * levelMultiplier),
		OwnerPlayerID: player.ID,
		DeployedTime:  time.Now(), // Mark deployment time
	}

	// Determine target tower based on Guard Tower 1 rule
	targetTower := h.findValidTarget(opponent)
	if targetTower != nil {
		troop.TargetID = targetTower.ID
		events = append(events, network.GameEventPayload{
			Message: fmt.Sprintf("%s's %s is initially targeting %s's %s", player.Username, troop.Name, opponent.Username, targetTower.Name),
			Time:    time.Now(),
		})
	} else {
		// No valid target at deployment time (shouldn't happen if King Tower exists)
		events = append(events, network.GameEventPayload{
			Message: fmt.Sprintf("%s's %s deployed but has no initial target.", player.Username, troop.Name),
			Time:    time.Now(),
		})
	}

	// Add troop to player's active troops and game board
	player.ActiveTroops[instanceID] = troop
	h.game.BoardState.ActiveTroops[instanceID] = troop

	return events, nil
}

// findValidTarget returns a valid tower target following the targeting rule:
// Guard Tower 1 must be destroyed before Guard Tower 2 or King Tower can be targeted.
// It targets the *lowest absolute HP* valid tower.
func (h *SimpleModeHandler) findValidTarget(opponent *PlayerInGame) *Tower {
	var guard1, guard2, king *Tower

	// Find the specific towers
	for _, t := range opponent.Towers {
		// Use SpecID for reliable identification, Name might have " 1" or " 2" appended
		switch t.SpecID {
		case "guard_tower":
			// Distinguish between Guard1 and Guard2 based on ID suffix or Name suffix
			// Assuming ID format "guard1_playerid" and "guard2_playerid"
			if t.ID == fmt.Sprintf("guard1_%s", opponent.ID) {
				guard1 = t
			} else if t.ID == fmt.Sprintf("guard2_%s", opponent.ID) {
				guard2 = t
			}
		case "king_tower":
			king = t
		}
	}

	// Targeting Logic:
	// 1. If Guard1 exists and has HP > 0, it MUST be the target.
	if guard1 != nil && guard1.CurrentHP > 0 {
		return guard1
	}

	// 2. If Guard1 is destroyed or doesn't exist:
	//    Target the lowest HP tower between Guard2 (if alive) and King Tower (if alive).
	var possibleTargets []*Tower
	if guard2 != nil && guard2.CurrentHP > 0 {
		possibleTargets = append(possibleTargets, guard2)
	}
	if king != nil && king.CurrentHP > 0 {
		possibleTargets = append(possibleTargets, king)
	}

	if len(possibleTargets) == 0 {
		return nil // No valid targets left
	}

	// Find the one with the lowest absolute HP among the valid targets
	lowestHPTarget := possibleTargets[0]
	for i := 1; i < len(possibleTargets); i++ {
		if possibleTargets[i].CurrentHP < lowestHPTarget.CurrentHP {
			lowestHPTarget = possibleTargets[i]
		}
	}
	return lowestHPTarget
}

// processTroopAttacks processes attacks from troops deployed in previous turns
func (h *SimpleModeHandler) processTroopAttacks(player *PlayerInGame, opponent *PlayerInGame) ([]network.GameEventPayload, error) {
	var events []network.GameEventPayload

	// Troops attack towers they're targeting
	for _, troop := range player.ActiveTroops {
		// Skip troops deployed this turn (they attack next turn)
		if time.Since(troop.DeployedTime) < time.Second*2 {
			continue
		}

		// Attack loop: Continue attacking as long as the troop is alive and has valid targets
		for troop.CurrentHP > 0 {
			// Find target tower
			targetTower, exists := h.game.BoardState.Towers[troop.TargetID]

			// If target doesn't exist or is already destroyed, find a new one
			if !exists || targetTower == nil || targetTower.CurrentHP <= 0 {
				newTarget := h.findValidTarget(opponent)
				if newTarget == nil {
					// No valid targets remain for this troop, break the attack loop
					troop.TargetID = "" // Clear target
					break
				}
				troop.TargetID = newTarget.ID
				targetTower = newTarget
				events = append(events, network.GameEventPayload{
					Message: fmt.Sprintf("%s's %s retargets %s's %s",
						player.Username, troop.Name, opponent.Username, targetTower.Name),
					Time: time.Now(),
				})
			}

			// Calculate damage using the combat function
			damage := CalculateDamage(troop.ATK, targetTower.DEF)

			// Apply damage
			targetTower.CurrentHP -= damage

			attackEvent := network.GameEventPayload{
				Message: fmt.Sprintf("%s's %s attacks %s's %s for %d damage (HP: %d/%d)",
					player.Username, troop.Name, opponent.Username, targetTower.Name, damage, targetTower.CurrentHP, targetTower.MaxHP),
				Time: time.Now(),
			}
			events = append(events, attackEvent)

			// Check if tower was destroyed
			if targetTower.CurrentHP <= 0 {
				targetTower.CurrentHP = 0 // Ensure HP doesn't go negative
				events = append(events, network.GameEventPayload{
					Message: fmt.Sprintf("%s's %s was destroyed!", opponent.Username, targetTower.Name),
					Time:    time.Now(),
				})

				// Tower destroyed, check for game over immediately
				gameOver, winnerIdx, loserIdx := h.checkGameOver()
				if gameOver {
					// If game over, stop further attacks by this troop
					events = append(events, network.GameEventPayload{
						Message: fmt.Sprintf("Game Over: %s wins! (King Tower destroyed)", h.game.Players[winnerIdx].Username),
						Time:    time.Now(),
					})
					h.game.GameState = GameStateFinished
					h.game.EndTime = time.Now()
					h.game.WinnerID = h.game.Players[winnerIdx].ID
					h.game.LoserID = h.game.Players[loserIdx].ID
					break
				}

				// Tower destroyed, but game not over. Find a new target and continue attacking in the same turn.
				continue
			} else {
				break
			}
		}
	}

	// Clean up defeated troops after all attacks are processed
	h.cleanupDefeatedUnits(player, opponent)

	return events, nil
}

// processTowerCounterattacks handles towers attacking troops that attacked them
func (h *SimpleModeHandler) processTowerCounterattacks(player *PlayerInGame, opponent *PlayerInGame) ([]network.GameEventPayload, error) {
	var events []network.GameEventPayload
	attackedTroopIDs := make(map[string]bool)

	// Iterate through opponent's towers to see if they need to counterattack
	for _, tower := range opponent.Towers {
		if tower.CurrentHP <= 0 || tower.TargetID == "" {
			continue
		}

		// Find the troop that last targeted this tower
		targetTroop, exists := h.game.BoardState.ActiveTroops[tower.TargetID]

		// Check if the troop still exists and belongs to the current player
		if !exists || targetTroop == nil || targetTroop.OwnerPlayerID != player.ID || targetTroop.CurrentHP <= 0 {
			continue
		}

		// Ensure a troop isn't counterattacked multiple times
		if attackedTroopIDs[targetTroop.InstanceID] {
			continue
		}

		// Calculate damage using the combat function
		damage := CalculateDamage(tower.ATK, targetTroop.DEF)

		// Apply damage
		targetTroop.CurrentHP -= damage
		attackedTroopIDs[targetTroop.InstanceID] = true

		events = append(events, network.GameEventPayload{
			Message: fmt.Sprintf("%s's %s counterattacks %s's %s for %d damage (HP: %d/%d)",
				opponent.Username, tower.Name, player.Username, targetTroop.Name, damage, targetTroop.CurrentHP, targetTroop.MaxHP),
			Time: time.Now(),
		})

		// Check if troop was defeated
		if targetTroop.CurrentHP <= 0 {
			targetTroop.CurrentHP = 0
			events = append(events, network.GameEventPayload{
				Message: fmt.Sprintf("%s's %s was defeated!", player.Username, targetTroop.Name),
				Time:    time.Now(),
			})
		}
	}

	// Clean up defeated troops after all attacks are processed
	h.cleanupDefeatedUnits(player, opponent)

	return events, nil
}

// cleanupDefeatedUnits removes troops and clears tower targets if the troop is defeated
func (h *SimpleModeHandler) cleanupDefeatedUnits(player1 *PlayerInGame, player2 *PlayerInGame) {
	// Check player 1's troops
	for id, troop := range player1.ActiveTroops {
		if troop.CurrentHP <= 0 {
			delete(player1.ActiveTroops, id)
			delete(h.game.BoardState.ActiveTroops, id)
			// Clear this troop as a target from any opponent tower
			for _, tower := range player2.Towers {
				if tower.TargetID == id {
					tower.TargetID = ""
				}
			}
		}
	}
	// Check player 2's troops
	for id, troop := range player2.ActiveTroops {
		if troop.CurrentHP <= 0 {
			delete(player2.ActiveTroops, id)
			delete(h.game.BoardState.ActiveTroops, id)
			// Clear this troop as a target from any opponent tower
			for _, tower := range player1.Towers {
				if tower.TargetID == id {
					tower.TargetID = ""
				}
			}
		}
	}
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
// Returns: bool (is game over?), winner index, loser index
func (h *SimpleModeHandler) checkGameOver() (bool, int, int) {
	// Check if either player's king tower is destroyed
	for playerIndex, player := range h.game.Players {
		kingTowerID := fmt.Sprintf("king_%s", player.ID)
		kingTower, exists := h.game.BoardState.Towers[kingTowerID]
		if exists && kingTower.CurrentHP <= 0 {
			// This player's king tower is destroyed, the other player wins
			winnerIndex := (playerIndex + 1) % 2
			loserIndex := playerIndex
			return true, winnerIndex, loserIndex
		}
	}

	return false, -1, -1 // Game not over
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
