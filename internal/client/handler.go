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
		printGameState(payload.InitialState, c.Username)

		return nil
	})

	// Handle state updates
	c.RegisterHandler(network.MessageTypeStateUpdate, func(msg *network.Message) error {
		var payload network.GameStatePayload
		if err := network.ParsePayload(msg, &payload); err != nil {
			return fmt.Errorf("failed to parse state update: %w", err)
		}

		fmt.Println("\nGame state updated:")
		printGameState(&payload, c.Username)

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
func printGameState(state *network.GameStatePayload, clientUsername string) {
	if state == nil {
		fmt.Println("No game state available")
		return
	}

	fmt.Println("\n====== TEXT CLASH ROYALE GAME BOARD ======")

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

	// Print board header
	if you != "" && opponent != "" {
		fmt.Printf("\n%-28s   VS   %28s\n", fmt.Sprintf("YOU (%s)", you), fmt.Sprintf("OPPONENT (%s)", opponent))
	} else if you != "" {
		fmt.Printf("\nYOU (%s)\n", you)
	} else if opponent != "" {
		fmt.Printf("\nOPPONENT (%s)\n", opponent)
	}
	fmt.Println(strings.Repeat("-", 60))

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

	// Draw towers side by side
	fmt.Println("TOWERS:")
	maxTowers := max(len(yourTowers), len(opponentTowers))
	for i := 0; i < maxTowers; i++ {
		yTowerStr := "                          "
		opTowerStr := "                          "

		if i < len(yourTowers) {
			t := yourTowers[i]
			hpPercent := float64(t.CurrentHP) / float64(t.MaxHP)
			hpBar := createHealthBar(hpPercent, 10)
			yTowerStr = fmt.Sprintf("%-8s %s %4d/%-4d", t.Position, hpBar, t.CurrentHP, t.MaxHP)
		}
		if i < len(opponentTowers) {
			t := opponentTowers[i]
			hpPercent := float64(t.CurrentHP) / float64(t.MaxHP)
			hpBar := createHealthBar(hpPercent, 10)
			opTowerStr = fmt.Sprintf("%-8s %s %4d/%-4d", t.Position, hpBar, t.CurrentHP, t.MaxHP)
		}
		fmt.Printf("%-28s | %-28s\n", yTowerStr, opTowerStr)
	}

	// Print active troops
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("ACTIVE TROOPS:")

	yourTroops := troopsByOwner[you]
	opponentTroops := troopsByOwner[opponent]

	if len(yourTroops) == 0 && len(opponentTroops) == 0 {
		fmt.Println("  (No active troops on either side)")
	} else {
		// Print your troops
		fmt.Printf("Your Troops (%s):\n", you)
		if len(yourTroops) == 0 {
			fmt.Println("  None")
		} else {
			for _, troop := range yourTroops {
				printTroopInfo(troop, opponentTowers)
			}
		}

		// Print opponent troops
		fmt.Printf("\nOpponent Troops (%s):\n", opponent)
		if len(opponentTroops) == 0 {
			fmt.Println("  None")
		} else {
			for _, troop := range opponentTroops {
				printTroopInfo(troop, yourTowers)
			}
		}
	}

	// Print mana info for Enhanced mode (if applicable)
	if state.YourMana > 0 || state.OpponentMana > 0 || state.TimeLeft > 0 {
		fmt.Println(strings.Repeat("-", 60))
		fmt.Println("ENHANCED MODE INFO:")
		fmt.Printf("  Your Mana: %d | Opponent Mana: %d | Time Left: %ds\n",
			state.YourMana, state.OpponentMana, state.TimeLeft)
	}

	fmt.Println("=========================================")
}

// printTroopInfo prints details for a single troop
func printTroopInfo(troop network.TroopInfo, opponentTowers []network.TowerInfo) {
	targetInfo := ""
	if troop.TargetTowerID != "" {
		// Find the target tower's name/position for better display
		targetName := troop.TargetTowerID // Fallback to ID
		for _, t := range opponentTowers {
			if t.ID == troop.TargetTowerID {
				targetName = t.Position // Use position (king, guard1, guard2)
				break
			}
		}
		targetInfo = fmt.Sprintf(" -> %s", targetName)
	}

	healthPercent := float64(troop.CurrentHP) / float64(troop.MaxHP)
	healthBar := createHealthBar(healthPercent, 10)

	fmt.Printf("  %-10s %s %4d/%-4d HP%s\n",
		troop.Name,
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

// Helper function (optional, could be inline)
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
