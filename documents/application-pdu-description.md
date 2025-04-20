
**Application PDU Description: Text-Based Clash Royale (TCR)**

**1. Introduction**

This document specifies the structure and content of the Application Protocol Data Units (PDUs) used for communication between the TCR Client and TCR Server. All communication occurs over a reliable TCP connection, and PDUs are encoded using the JSON format.

**2. General Message Format**

All PDUs exchanged between the client and server follow a consistent JSON structure:

```json
{
  "type": "UNIQUE_MESSAGE_TYPE_IDENTIFIER",
  "payload": {
    // Data specific to this message type
  }
}
```

*   `type` (string): A unique identifier indicating the purpose or type of the message. This field is mandatory for all PDUs.
*   `payload` (object): A JSON object containing the data associated with the message type. The structure of the payload varies depending on the `type`. This field is mandatory, but may be an empty object (`{}`) if no specific data is needed beyond the type identifier.

**3. Common Data Structures within Payloads**

Several common data structures are reused within the payloads of different PDU types:

*   **`TowerState`**: Represents the current state of a single tower.
    ```json
    {
      "id": "string", // Unique identifier for the tower (e.g., "player_king", "opponent_gt1")
      "owner": "player" | "opponent", // Perspective of the client receiving the state
      "spec_id": "string", // Identifier linking to base tower stats (e.g., "KingTower", "GuardTower")
      "max_hp": int, // Maximum HP (calculated based on base stats and owner's level)
      "current_hp": int // Current HP
    }
    ```
*   **`TroopState`**: Represents the current state of a single active troop on the board.
    ```json
    {
      "instance_id": "string", // Unique identifier for this specific troop instance (e.g., "pawn_123", "knight_456")
      "owner": "player" | "opponent", // Perspective of the client receiving the state
      "spec_id": "string", // Identifier linking to base troop stats (e.g., "Pawn", "Knight")
      "max_hp": int, // Maximum HP (calculated based on base stats and owner's level)
      "current_hp": int // Current HP
    }
    ```

**4. Client-to-Server (C2S) PDUs**

These messages are sent from the TCR Client to the TCR Server.

*   **`login_request`**: Sent by the client to authenticate with the server.
    ```json
    {
      "type": "login_request",
      "payload": {
        "username": "string",
        "password": "string"
      }
    }
    ```
*   **`deploy_troop_request`**: Sent by the client during their turn (Simple) or when affordable (Enhanced) to deploy a troop.
    ```json
    {
      "type": "deploy_troop_request",
      "payload": {
        "troop_id": "string" // The `spec_id` of the troop to deploy (e.g., "Pawn", "Queen")
      }
    }
    ```
*   **`quit_match_request`**: Sent by the client if they wish to forfeit the current match.
    ```json
    {
      "type": "quit_match_request",
      "payload": {}
    }
    ```

**5. Server-to-Client (S2C) PDUs**

These messages are sent from the TCR Server to the TCR Client(s).

*   **`login_response`**: Sent by the server in response to a `login_request`.
    ```json
    {
      "type": "login_response",
      "payload": {
        "success": bool, // True if authentication succeeded, false otherwise
        "message": "string" // Optional message, especially on failure (e.g., "Invalid credentials", "Already logged in")
      }
    }
    ```
*   **`match_found`**: Sent to two authenticated clients when they are paired for a game.
    ```json
    {
      "type": "match_found",
      "payload": {
        "opponent_username": "string" // The username of the opponent
      }
    }
    ```
*   **`game_start`**: Sent to both clients when the match begins, providing the initial game setup.
    ```json
    {
      "type": "game_start",
      "payload": {
        "mode": "Simple" | "Enhanced", // The game mode being played
        "player_level": int, // The client player's current level
        "opponent_level": int, // The opponent player's current level
        "initial_towers": [TowerState], // Array containing the initial state of all 6 towers
        // Enhanced Mode Only:
        "initial_mana": int // Starting MANA (typically 5)
      }
    }
    ```
*   **`game_state_update`**: Sent periodically (Enhanced) or after actions (Simple) to update the client's view of the game state.
    ```json
    {
      "type": "game_state_update",
      "payload": {
        "towers": [TowerState], // Current state of all towers
        "active_troops": [TroopState], // Current state of all active troops on the board
        // Enhanced Mode Only:
        "player_mana": int, // Client player's current MANA
        "opponent_mana": int, // Opponent player's current MANA (optional, maybe just show player's?)
        "time_left_seconds": int, // Seconds remaining in the match
        // Simple Mode Only:
        "current_turn": "player" | "opponent" // Whose turn it currently is
      }
    }
    ```
    *Note:* In Enhanced mode, this might be sent frequently (e.g., once per second). In Simple mode, it would typically be sent after each player's turn resolves.

*   **`game_event`**: Sent to inform the client about significant discrete events occurring in the game. Useful for displaying textual feedback or logs.
    ```json
    {
      "type": "game_event",
      "payload": {
        "message": "string" // Descriptive text of the event (e.g., "Opponent deployed a Knight!", "Your Guard Tower 1 was destroyed!", "Your Prince landed a CRITICAL HIT!", "Your King Tower healed for 300 HP!")
      }
    }
    ```
*   **`your_turn`**: (Simple Mode Only) Sent specifically to the client whose turn it is to act. Often sent just before or alongside a `game_state_update`.
    ```json
    {
      "type": "your_turn",
      "payload": {}
    }
    ```
*   **`game_over`**: Sent to both clients when the game concludes.
    ```json
    {
      "type": "game_over",
      "payload": {
        "outcome": "win" | "loss" | "draw", // Result from the client's perspective
        "reason": "string", // Explanation (e.g., "Opponent's King Tower destroyed", "Your King Tower destroyed", "Timeout - More towers destroyed", "Timeout - Less towers destroyed", "Timeout - Equal towers", "Opponent quit")
        // Enhanced Mode Only (or if EXP is added to Simple):
        "exp_earned": int, // Total EXP gained from this match (tower destruction + win/draw bonus)
        "new_total_exp": int, // Player's total EXP after the match
        "level_up": bool // True if the player leveled up as a result of this match
      }
    }
    ```
*   **`error_response`**: Sent by the server if the client sends an invalid request or tries an illegal action.
    ```json
    {
      "type": "error_response",
      "payload": {
        "message": "string" // Description of the error (e.g., "Invalid command", "Not enough MANA", "Not your turn", "Invalid troop ID")
      }
    }
    ```

