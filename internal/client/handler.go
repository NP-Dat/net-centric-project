package client

import (
	"fmt"
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

	// Print towers
	fmt.Println("Towers:")
	for _, tower := range state.Towers {
		fmt.Printf("  %s's %s: %d/%d HP\n", tower.OwnerUsername, tower.Name, tower.CurrentHP, tower.MaxHP)
	}

	// Print troops
	fmt.Println("Active Troops:")
	if len(state.Troops) == 0 {
		fmt.Println("  No active troops")
	} else {
		for _, troop := range state.Troops {
			targetInfo := ""
			if troop.TargetTowerID != "" {
				targetInfo = fmt.Sprintf(" (targeting %s)", troop.TargetTowerID)
			}
			fmt.Printf("  %s's %s: %d/%d HP%s\n", troop.OwnerUsername, troop.Name, troop.CurrentHP, troop.MaxHP, targetInfo)
		}
	}

	// Print mana info for Enhanced mode
	if state.YourMana > 0 {
		fmt.Printf("Your Mana: %d\n", state.YourMana)
		fmt.Printf("Opponent's Mana: %d\n", state.OpponentMana)
		fmt.Printf("Time left: %d seconds\n", state.TimeLeft)
	}
}
