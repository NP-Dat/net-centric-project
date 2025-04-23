package client

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/NP-Dat/net-centric-project/internal/network"
)

// SetupDefaultHandlers sets up the default message handlers for the client
func (c *Client) SetupDefaultHandlers() {
	// Handle authentication results
	c.RegisterHandler(network.MessageTypeAuthResult, func(msg *network.Message) error {
		var payload network.AuthResultPayload
		if err := network.ParsePayload(msg, &payload); err != nil {
			return fmt.Errorf("failed to parse auth result: %w", err)
		}

		if payload.Success {
			fmt.Printf("Authentication successful. Welcome, %s!\n", c.Username)
		} else {
			fmt.Printf("Authentication failed: %s\n", payload.Message)
		}

		return nil
	})

	// Handle game events
	c.RegisterHandler(network.MessageTypeGameEvent, func(msg *network.Message) error {
		var payload network.GameEventPayload
		if err := network.ParsePayload(msg, &payload); err != nil {
			return fmt.Errorf("failed to parse game event: %w", err)
		}

		timeStr := payload.Time.Format(time.RFC3339)
		fmt.Printf("[%s] Event: %s\n", timeStr, payload.Message)

		return nil
	})

	// Handle game start
	c.RegisterHandler(network.MessageTypeGameStart, func(msg *network.Message) error {
		var payload network.GameStartPayload
		if err := network.ParsePayload(msg, &payload); err != nil {
			return fmt.Errorf("failed to parse game start: %w", err)
		}

		fmt.Printf("\nGame started!\n")
		fmt.Printf("Game ID: %s\n", payload.GameID)
		fmt.Printf("Opponent: %s\n", payload.OpponentUsername)
		fmt.Printf("Game Mode: %s\n", payload.GameMode)

		if payload.GameMode == "simple" {
			if payload.YourTurn {
				fmt.Println("It's your turn!")
			} else {
				fmt.Println("Waiting for opponent's turn...")
			}
		}

		fmt.Println("\nInitial game state:")
		printGameState(payload.InitialState)

		return nil
	})

	// Handle state updates
	c.RegisterHandler(network.MessageTypeStateUpdate, func(msg *network.Message) error {
		var payload network.GameStatePayload
		if err := network.ParsePayload(msg, &payload); err != nil {
			return fmt.Errorf("failed to parse state update: %w", err)
		}

		fmt.Println("\nGame state updated:")
		printGameState(&payload)

		return nil
	})

	// Handle turn changes
	c.RegisterHandler(network.MessageTypeTurnChange, func(msg *network.Message) error {
		var payload network.TurnChangePayload
		if err := network.ParsePayload(msg, &payload); err != nil {
			return fmt.Errorf("failed to parse turn change: %w", err)
		}

		if payload.YourTurn {
			fmt.Println("\nIt's your turn now!")
		} else {
			fmt.Println("\nIt's your opponent's turn now.")
		}

		return nil
	})

	// Handle game over
	c.RegisterHandler(network.MessageTypeGameOver, func(msg *network.Message) error {
		var payload network.GameOverPayload
		if err := network.ParsePayload(msg, &payload); err != nil {
			return fmt.Errorf("failed to parse game over: %w", err)
		}

		fmt.Println("\nGame Over!")
		fmt.Printf("Reason: %s\n", payload.Reason)

		if payload.Winner == c.Username {
			fmt.Println("You win!")
		} else if payload.Winner == "" {
			fmt.Println("It's a draw!")
		} else {
			fmt.Printf("Winner: %s\n", payload.Winner)
		}

		fmt.Printf("EXP earned: %d\n", payload.ExpEarned)
		fmt.Printf("Total EXP: %d\n", payload.NewTotalExp)
		fmt.Printf("Current level: %d\n", payload.NewLevel)

		if payload.LeveledUp {
			fmt.Println("Congratulations! You leveled up!")
		}

		return nil
	})
}

// printGameState prints the current game state
func printGameState(state *network.GameStatePayload) {
	if state == nil {
		fmt.Println("No game state available")
		return
	}

	// Draw a visual representation of the game board
	fmt.Println("\n====== TEXT CLASH ROYALE GAME BOARD ======")

	// Group towers by username for easier display
	playerTowers := make(map[string][]network.TowerInfo)
	for _, tower := range state.Towers {
		playerTowers[tower.OwnerUsername] = append(playerTowers[tower.OwnerUsername], tower)
	}

	// Print board header with player names
	var players []string
	for player := range playerTowers {
		players = append(players, player)
	}

	if len(players) >= 2 {
		fmt.Printf("\n%s %40s\n", players[0], players[1])
		fmt.Println(strings.Repeat("-", 60))

		// Find player towers
		player1Towers := playerTowers[players[0]]
		player2Towers := playerTowers[players[1]]

		// Sort towers by position
		sortTowersByPosition := func(towers []network.TowerInfo) []network.TowerInfo {
			positionPriority := map[string]int{"guard1": 0, "guard2": 1, "king": 2}
			sort.Slice(towers, func(i, j int) bool {
				return positionPriority[towers[i].Position] < positionPriority[towers[j].Position]
			})
			return towers
		}

		player1Towers = sortTowersByPosition(player1Towers)
		player2Towers = sortTowersByPosition(player2Towers)

		// Draw towers side by side
		fmt.Println("TOWERS:")
		for i := 0; i < 3 && i < len(player1Towers) && i < len(player2Towers); i++ {
			p1Tower := player1Towers[i]
			p2Tower := player2Towers[i]

			// Create health bar representations
			p1HealthPercent := float64(p1Tower.CurrentHP) / float64(p1Tower.MaxHP)
			p2HealthPercent := float64(p2Tower.CurrentHP) / float64(p2Tower.MaxHP)

			p1HealthBar := createHealthBar(p1HealthPercent, 10)
			p2HealthBar := createHealthBar(p2HealthPercent, 10)

			fmt.Printf("%-10s %s %4d/%-4d HP %15s %s %4d/%-4d HP\n",
				p1Tower.Position,
				p1HealthBar,
				p1Tower.CurrentHP,
				p1Tower.MaxHP,
				p2Tower.Position,
				p2HealthBar,
				p2Tower.CurrentHP,
				p2Tower.MaxHP)
		}
	}

	// Print active troops
	fmt.Println("\nACTIVE TROOPS:")
	if len(state.Troops) == 0 {
		fmt.Println("  No active troops")
	} else {
		// Group troops by owner
		troopsByOwner := make(map[string][]network.TroopInfo)
		for _, troop := range state.Troops {
			troopsByOwner[troop.OwnerUsername] = append(troopsByOwner[troop.OwnerUsername], troop)
		}

		// Print troops for each player
		for player, troops := range troopsByOwner {
			fmt.Printf("\n%s's Troops:\n", player)
			for _, troop := range troops {
				targetInfo := ""
				if troop.TargetTowerID != "" {
					// Extract just the position part of the tower ID for clarity
					parts := strings.Split(troop.TargetTowerID, "_")
					if len(parts) > 1 {
						targetInfo = fmt.Sprintf(" → targeting %s", parts[1])
					} else {
						targetInfo = fmt.Sprintf(" → targeting %s", troop.TargetTowerID)
					}
				}

				// Create health bar
				healthPercent := float64(troop.CurrentHP) / float64(troop.MaxHP)
				healthBar := createHealthBar(healthPercent, 10)

				fmt.Printf("  %-10s %s %4d/%-4d HP%s\n",
					troop.Name,
					healthBar,
					troop.CurrentHP,
					troop.MaxHP,
					targetInfo)
			}
		}
	}

	// Print mana info for Enhanced mode
	if state.YourMana > 0 {
		fmt.Println("\nMANA:")
		fmt.Printf("Your Mana: %d\n", state.YourMana)
		fmt.Printf("Opponent's Mana: %d\n", state.OpponentMana)
		fmt.Printf("Time left: %d seconds\n", state.TimeLeft)
	}

	fmt.Println("\n=========================================")
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
		bar = fmt.Sprintf("[%s%s]", strings.Repeat("#", filledLength), strings.Repeat("-", emptyLength))
	} else if percent > 0.3 {
		// Yellow for medium health
		bar = fmt.Sprintf("[%s%s]", strings.Repeat("=", filledLength), strings.Repeat("-", emptyLength))
	} else {
		// Red for low health
		bar = fmt.Sprintf("[%s%s]", strings.Repeat("!", filledLength), strings.Repeat("-", emptyLength))
	}

	return bar
}

// Interactive login prompt for user authentication
func (c *Client) PromptLogin() error {
	var username, password string

	fmt.Println("=== Text Clash Royale Login ===")
	fmt.Print("Username: ")
	fmt.Scanln(&username)

	fmt.Print("Password: ")
	fmt.Scanln(&password)

	// Validate inputs
	if username == "" || password == "" {
		return fmt.Errorf("username and password cannot be empty")
	}

	// Attempt to login with provided credentials
	return c.LoginWithCredentials(username, password)
}

// SendMessage sends a chat message to the server
func (c *Client) SendMessage(message string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to server")
	}

	// For now, we'll use GameEvent for messages during Sprint 1
	// In later sprints, we might implement a dedicated chat system
	messagePayload := &network.GameEventPayload{
		Message: message,
		Time:    time.Now(),
	}

	return c.Send(network.MessageTypeGameEvent, messagePayload)
}

// DeployTroop sends a request to deploy a troop
func (c *Client) DeployTroop(troopID string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to server")
	}

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

	switch command {
	case "login":
		// Handle login command
		if len(args) != 2 {
			return fmt.Errorf("usage: login <username> <password>")
		}
		return c.LoginWithCredentials(args[0], args[1])

	case "join":
		// Join matchmaking queue
		return c.JoinMatchmaking()

	case "deploy":
		// Deploy a troop
		if len(args) != 1 {
			return fmt.Errorf("usage: deploy <troop_id>")
		}
		return c.DeployTroop(args[0])

	case "quit":
		// Quit the game/connection
		return c.Disconnect()

	case "help":
		// Display available commands
		fmt.Println("\nAvailable Commands:")
		fmt.Println("  login <username> <password> - Log in to the server")
		fmt.Println("  join - Join the matchmaking queue")
		fmt.Println("  deploy <troop_id> - Deploy a troop (pawn, bishop, rook, knight, prince, queen)")
		fmt.Println("  quit - Disconnect from the server")
		fmt.Println("  help - Display this help message")
		return nil

	default:
		// Treat as a chat message for now
		return c.SendMessage(input)
	}
}
