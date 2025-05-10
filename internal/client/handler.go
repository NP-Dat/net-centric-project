package client

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/NP-Dat/net-centric-project/internal/network"
	"github.com/NP-Dat/net-centric-project/pkg/logger"
)

// SetupDefaultHandlers sets up the default message handlers for the client
func (c *Client) SetupDefaultHandlers() {
	// Handle authentication results
	c.RegisterHandler(network.MessageTypeAuthResult, func(msg *network.Message) error {
		var payload network.AuthResultPayload
		if err := network.ParsePayload(msg, &payload); err != nil {
			logger.Client.Error("Failed to parse authentication result: %v", err)
			fmt.Printf("Error: Could not process authentication response\n")
			return err
		}

		if payload.Success {
			logger.Client.Info("Authentication successful for user: %s", c.Username)
			fmt.Printf("\n✓ Authentication successful. Welcome, %s!\n", c.Username)
		} else {
			logger.Client.Warn("Authentication failed for user: %s - %s", c.Username, payload.Message)
			fmt.Printf("\n❌ Authentication failed: %s\n", payload.Message)
		}

		return nil
	})

	// Handle game events
	c.RegisterHandler(network.MessageTypeGameEvent, func(msg *network.Message) error {
		var payload network.GameEventPayload
		if err := network.ParsePayload(msg, &payload); err != nil {
			logger.Client.Error("Failed to parse game event: %v", err)
			fmt.Printf("Error: Could not process game event\n")
			return err
		}

		timeStr := payload.Time.Format("15:04:05")
		logger.Client.Debug("Game event received: %s", payload.Message)

		// Enhance the event display
		eventMsg := payload.Message

		// Format system messages differently than player messages
		if strings.HasPrefix(eventMsg, "[") && strings.Contains(eventMsg, "]: ") {
			// This is likely a player message, keep the original format
			fmt.Printf("[%s] %s\n", timeStr, eventMsg)
		} else {
			// This is a system message
			fmt.Printf("[%s] 📢 %s\n", timeStr, eventMsg)
		}

		return nil
	})

	// Handle game start
	c.RegisterHandler(network.MessageTypeGameStart, func(msg *network.Message) error {
		var payload network.GameStartPayload
		if err := network.ParsePayload(msg, &payload); err != nil {
			logger.Client.Error("Failed to parse game start: %v", err)
			fmt.Printf("Error: Could not process game start\n")
			return err
		}

		logger.Client.Info("Game started - ID: %s, Opponent: %s, Mode: %s",
			payload.GameID, payload.OpponentUsername, payload.GameMode)

		// Enhanced game start notification
		fmt.Printf("\n🎮 === GAME STARTED! === 🎮\n")
		fmt.Printf("Game ID: %s\n", payload.GameID)
		fmt.Printf("Opponent: %s\n", payload.OpponentUsername)
		fmt.Printf("Mode: %s\n", payload.GameMode)

		if payload.GameMode == "simple" {
			if payload.YourTurn {
				fmt.Println("\n➤ It's your turn to play!")
				fmt.Println("  Use 'deploy <troop>' to deploy a troop (pawn, bishop, rook, knight, prince, queen)")
			} else {
				fmt.Println("\n⏳ Waiting for opponent's turn...")
			}
		}

		fmt.Println("\n===== INITIAL GAME STATE =====")
		printGameState(payload.InitialState, c.Username)

		return nil
	})

	// Handle state updates
	c.RegisterHandler(network.MessageTypeStateUpdate, func(msg *network.Message) error {
		var payload network.GameStatePayload
		if err := network.ParsePayload(msg, &payload); err != nil {
			logger.Client.Error("Failed to parse state update: %v", err)
			fmt.Printf("Error: Could not process game state update\n")
			return err
		}

		logger.Client.Debug("Game state updated - Troops: %d, Towers: %d",
			len(payload.Troops), len(payload.Towers))

		fmt.Println("\n===== GAME STATE UPDATED =====")
		printGameState(&payload, c.Username)

		return nil
	})

	// Handle turn changes
	c.RegisterHandler(network.MessageTypeTurnChange, func(msg *network.Message) error {
		var payload network.TurnChangePayload
		if err := network.ParsePayload(msg, &payload); err != nil {
			logger.Client.Error("Failed to parse turn change: %v", err)
			fmt.Printf("Error: Could not process turn change\n")
			return err
		}

		logger.Client.Debug("Turn changed - Your turn: %v", payload.YourTurn)

		if payload.YourTurn {
			fmt.Println("\n➤ It's your turn now!")
			fmt.Println("  Use 'deploy <troop>' to deploy a troop (pawn, bishop, rook, knight, prince, queen)")
		} else {
			fmt.Println("\n⏳ It's your opponent's turn now.")
		}

		return nil
	})

	// Handle game over
	c.RegisterHandler(network.MessageTypeGameOver, func(msg *network.Message) error {
		var payload network.GameOverPayload
		if err := network.ParsePayload(msg, &payload); err != nil {
			logger.Client.Error("Failed to parse game over: %v", err)
			fmt.Printf("Error: Could not process game over\n")
			return err
		}

		logger.Client.Info("Game over - Winner: %s, Reason: %s, EXP earned: %d",
			payload.Winner, payload.Reason, payload.ExpEarned)

		fmt.Println("\n🏁 ========= GAME OVER ========= 🏁")
		fmt.Printf("Reason: %s\n", payload.Reason)

		if payload.Winner == c.Username {
			fmt.Println("\n🏆 You win! 🏆")
		} else if payload.Winner == "" {
			fmt.Println("\n🤝 It's a draw! 🤝")
		} else {
			fmt.Printf("\n💔 %s wins the game\n", payload.Winner)
		}

		fmt.Printf("\n📊 Game Stats:\n")
		fmt.Printf("  ⭐ EXP earned: %d\n", payload.ExpEarned)
		fmt.Printf("  ⭐ Total EXP: %d\n", payload.NewTotalExp)
		fmt.Printf("  ⭐ Current level: %d\n", payload.NewLevel)

		if payload.LeveledUp {
			fmt.Println("\n🎉 CONGRATULATIONS! You leveled up! 🎉")
		}

		fmt.Println("\n=================================")
		fmt.Println("Type 'join' to queue for a new game or 'quit' to exit")

		return nil
	})

	// Handle errors
	c.RegisterHandler(network.MessageTypeError, func(msg *network.Message) error {
		var payload network.ErrorPayload
		if err := network.ParsePayload(msg, &payload); err != nil {
			logger.Client.Error("Failed to parse error message: %v", err)
			fmt.Printf("Error: Received an error message but couldn't read it\n")
			return err
		}

		logger.Client.Warn("Error message from server: [%d] %s", payload.Code, payload.Message)
		fmt.Printf("\n❌ Server Error [%d]: %s\n", payload.Code, payload.Message)

		return nil
	})

	// Handle troop choices from server
	c.RegisterHandler(network.MessageTypeTroopChoices, func(msg *network.Message) error {
		var payload network.TroopChoicesPayload
		if err := network.ParsePayload(msg, &payload); err != nil {
			logger.Client.Error("Failed to parse troop choices: %v", err)
			fmt.Printf("Error: Could not process troop choices from server\n")
			return err
		}

		c.SetCurrentTroopChoices(payload.Choices) // Store choices in client

		if len(payload.Choices) == 0 {
			fmt.Println("\n📋 No troops available to deploy this turn.")
			// If it's our turn, this is unusual, might mean game logic issue or all troops used up
			// Or perhaps it implies skipping deployment phase.
			// For now, just inform. Client might need to send a "skip" or "no_action" if applicable.
			return nil
		}

		fmt.Println("\n📋 Choose a troop to deploy:")
		for i, choice := range payload.Choices {
			fmt.Printf("  %d. %s (ID: %s, Mana: %d)\n", i+1, choice.Name, choice.ID, choice.ManaCost)
		}
		fmt.Println("  Use 'deploy <troop_id_or_number>' to deploy.") // Update prompt guidance
		// Example: deploy pawn OR deploy 1

		return nil
	})

	logger.Client.Info("Default message handlers set up")
}

// printGameState prints the current game state
func printGameState(state *network.GameStatePayload, clientUsername string) {
	if state == nil {
		logger.Client.Warn("Attempted to print nil game state")
		fmt.Println("No game state available")
		return
	}

	// Group towers and troops by username
	playerTowers := make(map[string][]network.TowerInfo)
	troopsByOwner := make(map[string][]network.TroopInfo)
	var playerUsernames []string
	playerMap := make(map[string]bool)

	for _, tower := range state.Towers {
		playerTowers[tower.OwnerUsername] = append(playerTowers[tower.OwnerUsername], tower)
		if !playerMap[tower.OwnerUsername] {
			playerUsernames = append(playerUsernames, tower.OwnerUsername)
			playerMap[tower.OwnerUsername] = true
		}
	}
	for _, troop := range state.Troops {
		troopsByOwner[troop.OwnerUsername] = append(troopsByOwner[troop.OwnerUsername], troop)
		if !playerMap[troop.OwnerUsername] {
			playerUsernames = append(playerUsernames, troop.OwnerUsername)
			playerMap[troop.OwnerUsername] = true
		}
	}

	// Determine which player is 'you' and which is 'opponent'
	var you, opponent string
	if len(playerUsernames) > 0 {
		if playerUsernames[0] == clientUsername {
			you = playerUsernames[0]
			if len(playerUsernames) > 1 {
				opponent = playerUsernames[1]
			}
		} else {
			opponent = playerUsernames[0]
			if len(playerUsernames) > 1 {
				you = playerUsernames[1]
			}
		}
	}

	// Print stylized game board header
	fmt.Println("\n╔══════════════ TEXT CLASH ROYALE ══════════════╗")

	// Print board header with player information
	if you != "" && opponent != "" {
		fmt.Printf("║ %-23s  VS  %-23s ║\n",
			fmt.Sprintf("YOU (%s)", you),
			fmt.Sprintf("OPPONENT (%s)", opponent))
	} else if you != "" {
		fmt.Printf("║ YOU (%s) %-40s ║\n", you, "")
	} else if opponent != "" {
		fmt.Printf("║ OPPONENT (%s) %-36s ║\n", opponent, "")
	}
	fmt.Println("╟─────────────────────────────────────────────╢")

	// Get tower lists
	yourTowers := playerTowers[you]
	opponentTowers := playerTowers[opponent]

	// Sort towers by position (Guard1, Guard2, King)
	sortTowersByPosition := func(towers []network.TowerInfo) []network.TowerInfo {
		positionPriority := map[string]int{"guard1": 0, "guard2": 1, "king": 2, "unknown": 3}
		sort.Slice(towers, func(i, j int) bool {
			return positionPriority[towers[i].Position] < positionPriority[towers[j].Position]
		})
		return towers
	}

	yourTowers = sortTowersByPosition(yourTowers)
	opponentTowers = sortTowersByPosition(opponentTowers)

	// Draw towers section header
	fmt.Println("║                    TOWERS                    ║")

	// Draw towers side by side with improved formatting
	maxTowers := max(len(yourTowers), len(opponentTowers))
	for i := 0; i < maxTowers; i++ {
		yTowerStr := "                      "
		opTowerStr := "                      "

		if i < len(yourTowers) {
			t := yourTowers[i]
			hpPercent := float64(t.CurrentHP) / float64(t.MaxHP)
			hpBar := createHealthBar(hpPercent, 8)

			// Format tower name more clearly
			towerName := strings.Title(t.Position)
			if strings.HasPrefix(strings.ToLower(t.Position), "guard") {
				num := strings.TrimPrefix(strings.ToLower(t.Position), "guard")
				towerName = fmt.Sprintf("Guard #%s", num)
			} else if strings.ToLower(t.Position) == "king" {
				towerName = "King Tower"
			}

			yTowerStr = fmt.Sprintf("%-10s %s %4d/%-4d", towerName, hpBar, t.CurrentHP, t.MaxHP)
		}
		if i < len(opponentTowers) {
			t := opponentTowers[i]
			hpPercent := float64(t.CurrentHP) / float64(t.MaxHP)
			hpBar := createHealthBar(hpPercent, 8)

			// Format tower name more clearly
			towerName := strings.Title(t.Position)
			if strings.HasPrefix(strings.ToLower(t.Position), "guard") {
				num := strings.TrimPrefix(strings.ToLower(t.Position), "guard")
				towerName = fmt.Sprintf("Guard #%s", num)
			} else if strings.ToLower(t.Position) == "king" {
				towerName = "King Tower"
			}

			opTowerStr = fmt.Sprintf("%-10s %s %4d/%-4d", towerName, hpBar, t.CurrentHP, t.MaxHP)
		}
		fmt.Printf("║ %-24s│ %-24s ║\n", yTowerStr, opTowerStr)
	}

	// Divider between towers and troops
	fmt.Println("╟─────────────────────────────────────────────╢")

	// Print active troops section
	fmt.Println("║                 ACTIVE TROOPS                ║")

	yourTroops := troopsByOwner[you]
	opponentTroops := troopsByOwner[opponent]

	if len(yourTroops) == 0 && len(opponentTroops) == 0 {
		fmt.Println("║          (No active troops on either side)     ║")
	} else {
		// Print your troops header with clearer formatting
		fmt.Printf("║ YOUR TROOPS:%-34s║\n", "")

		// Print your troops or "None"
		if len(yourTroops) == 0 {
			fmt.Println("║   None                                      ║")
		} else {
			for _, troop := range yourTroops {
				troopInfo := formatTroopInfo(troop, opponentTowers)
				fmt.Printf("║   %-43s ║\n", troopInfo)
			}
		}

		// Print opponent troops header
		fmt.Printf("║ OPPONENT TROOPS:%-30s║\n", "")

		// Print opponent troops or "None"
		if len(opponentTroops) == 0 {
			fmt.Println("║   None                                      ║")
		} else {
			for _, troop := range opponentTroops {
				troopInfo := formatTroopInfo(troop, yourTowers)
				fmt.Printf("║   %-43s ║\n", troopInfo)
			}
		}
	}

	// Print mana info for Enhanced mode (if applicable) with better formatting
	if state.YourMana > 0 || state.OpponentMana > 0 || state.TimeLeft > 0 {
		fmt.Println("╟─────────────────────────────────────────────╢")
		fmt.Println("║             ENHANCED MODE INFO              ║")
		fmt.Printf("║  Your Mana: %-2d | Opponent Mana: %-2d | Time: %-3ds ║\n",
			state.YourMana, state.OpponentMana, state.TimeLeft)
	}

	fmt.Println("╚═════════════════════════════════════════════╝")
}

// formatTroopInfo formats a single troop's info for display
func formatTroopInfo(troop network.TroopInfo, opponentTowers []network.TowerInfo) string {
	targetInfo := ""
	if troop.TargetTowerID != "" {
		// Find the target tower's name/position for better display
		targetName := troop.TargetTowerID // Fallback to ID
		for _, t := range opponentTowers {
			if t.ID == troop.TargetTowerID {
				// Format tower position name nicely
				if strings.HasPrefix(strings.ToLower(t.Position), "guard") {
					num := strings.TrimPrefix(strings.ToLower(t.Position), "guard")
					targetName = fmt.Sprintf("Guard #%s", num)
				} else if strings.ToLower(t.Position) == "king" {
					targetName = "King Tower"
				} else {
					targetName = strings.Title(t.Position)
				}
				break
			}
		}
		targetInfo = fmt.Sprintf(" → %s", targetName)
	}

	healthPercent := float64(troop.CurrentHP) / float64(troop.MaxHP)
	healthBar := createHealthBar(healthPercent, 8)

	return fmt.Sprintf("%-7s %s %4d/%-4d HP%s",
		strings.Title(troop.Name),
		healthBar,
		troop.CurrentHP,
		troop.MaxHP,
		targetInfo)
}

// createHealthBar generates a visual health bar based on percentage
func createHealthBar(percent float64, length int) string {
	if percent < 0 {
		percent = 0
	} else if percent > 1 {
		percent = 1
	}

	filledLength := int(percent * float64(length))
	emptyLength := length - filledLength

	var bar string
	if percent > 0.7 {
		// Green for high health
		bar = fmt.Sprintf("[%s%s]", strings.Repeat("█", filledLength), strings.Repeat("▒", emptyLength))
	} else if percent > 0.3 {
		// Yellow for medium health
		bar = fmt.Sprintf("[%s%s]", strings.Repeat("▓", filledLength), strings.Repeat("▒", emptyLength))
	} else {
		// Red for low health
		bar = fmt.Sprintf("[%s%s]", strings.Repeat("▒", filledLength), strings.Repeat("░", emptyLength))
	}

	return bar
}

// Interactive login prompt for user authentication
func (c *Client) PromptLogin() error {
	var username, password string

	fmt.Println("\n╔═══════════════════════════════╗")
	fmt.Println("║    TEXT CLASH ROYALE LOGIN    ║")
	fmt.Println("╚═══════════════════════════════╝")

	fmt.Print("Username: ")
	fmt.Scanln(&username)

	fmt.Print("Password: ")
	fmt.Scanln(&password)

	// Validate inputs
	if username == "" || password == "" {
		logger.Client.Warn("Login attempt with empty username or password")
		fmt.Println("\n❌ Username and password cannot be empty")
		return fmt.Errorf("username and password cannot be empty")
	}

	logger.Client.Info("Login attempt with username: %s", username)

	// Attempt to login with provided credentials
	return c.LoginWithCredentials(username, password)
}

// SendMessage sends a chat message to the server
func (c *Client) SendMessage(message string) error {
	if !c.IsConnected() {
		logger.Client.Error("Attempted to send message while not connected")
		return fmt.Errorf("not connected to server")
	}

	// For now, we'll use GameEvent for messages during Sprint 1
	messagePayload := &network.GameEventPayload{
		Message: message,
		Time:    time.Now(),
	}

	logger.Client.Debug("Sending chat message: %s", message)
	return c.Send(network.MessageTypeGameEvent, messagePayload)
}

// DeployTroop sends a request to deploy a troop
func (c *Client) DeployTroop(troopID string) error {
	if !c.IsConnected() {
		logger.Client.Error("Attempted to deploy troop while not connected")
		return fmt.Errorf("not connected to server")
	}

	logger.Client.Info("Attempting to deploy troop: %s", troopID)

	deployPayload := &network.DeployTroopPayload{
		TroopID: troopID,
		// TargetTowerID is optional and can be omitted as targeting is implicit in Simple mode
	}

	return c.Send(network.MessageTypeDeployTroop, deployPayload)
}

// ParseCommand parses and handles client commands
func (c *Client) ParseCommand(input string) error {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	// Split the input into command and arguments
	parts := strings.Fields(input)
	command := strings.ToLower(parts[0])
	args := parts[1:]

	logger.Client.Debug("Command received: %s with %d arguments", command, len(args))

	switch command {
	case "login":
		// Handle login command
		if len(args) != 2 {
			logger.Client.Warn("Invalid login command format")
			fmt.Println("\n❌ Usage: login <username> <password>")
			return fmt.Errorf("usage: login <username> <password>")
		}
		return c.LoginWithCredentials(args[0], args[1])

	case "join":
		// Join matchmaking queue
		logger.Client.Info("User requested to join matchmaking")
		fmt.Println("\n⌛ Requesting to join matchmaking queue...")
		return c.JoinMatchmaking()

	case "deploy":
		// Deploy a troop
		if len(args) != 1 {
			logger.Client.Warn("Invalid deploy command format")
			fmt.Println("\n❌ Usage: deploy <troop_id>")
			fmt.Println("Available troops: pawn, bishop, rook, knight, prince, queen")
			return fmt.Errorf("usage: deploy <troop_id>")
		}

		troopID := strings.ToLower(args[0])

		// Validate troop type
		validTroops := map[string]bool{
			"pawn": true, "bishop": true, "rook": true,
			"knight": true, "prince": true, "queen": true,
		}

		if !validTroops[troopID] {
			logger.Client.Warn("Invalid troop type: %s", troopID)
			fmt.Printf("\n❌ Invalid troop: '%s'\n", troopID)
			fmt.Println("Available troops: pawn, bishop, rook, knight, prince, queen")
			return fmt.Errorf("invalid troop type: %s", troopID)
		}

		fmt.Printf("\n⚔️ Deploying %s...\n", strings.Title(troopID))
		return c.DeployTroop(troopID)

	case "quit":
		// Quit the game/connection
		logger.Client.Info("User requested to quit")
		fmt.Println("\n👋 Disconnecting from server...")
		return c.Disconnect()

	case "help":
		// Display available commands with enhanced formatting
		logger.Client.Debug("Help command requested")
		fmt.Println("\n╔═════════════ AVAILABLE COMMANDS ═════════════╗")
		fmt.Println("║                                               ║")
		fmt.Println("║  login <username> <password>                  ║")
		fmt.Println("║    Log in to the server                       ║")
		fmt.Println("║                                               ║")
		fmt.Println("║  join                                         ║")
		fmt.Println("║    Join the matchmaking queue                 ║")
		fmt.Println("║                                               ║")
		fmt.Println("║  deploy <troop>                               ║")
		fmt.Println("║    Deploy a troop in the current game         ║")
		fmt.Println("║    Available troops: pawn, bishop, rook,      ║")
		fmt.Println("║                      knight, prince, queen    ║")
		fmt.Println("║                                               ║")
		fmt.Println("║  quit                                         ║")
		fmt.Println("║    Disconnect from the server                 ║")
		fmt.Println("║                                               ║")
		fmt.Println("║  help                                         ║")
		fmt.Println("║    Display this help message                  ║")
		fmt.Println("║                                               ║")
		fmt.Println("╚═══════════════════════════════════════════════╝")
		return nil

	default:
		// Treat as a chat message
		logger.Client.Debug("Treating input as chat message: %s", input)
		return c.SendMessage(input)
	}
}

// Helper function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
