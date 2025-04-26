**Phase 1: Understanding & Definition (Updated)**

*   **Goal:** Create a text-based, client-server version of a Clash Royale-like game using Go.
*   **Core Architecture:** Client-Server model.
*   **Network Protocol:** TCP exclusively for reliable communication.
*   **Data Serialization:** JSON for client-server messages and data persistence.
*   **Authentication:** Users connect with a username and password. Passwords must be securely hashed (bcrypt) on the server before storage.
*   **Persistence:**
    *   Player data (Username, HashedPassword, EXP, Level) stored in JSON files (e.g., `data/players/username.json`).
    *   Base Tower and Troop specifications (including HP, ATK, DEF, CRIT% for Towers, Mana Cost for Troops) loaded from configuration JSON files (`config/towers.json`, `config/troops.json`).
*   **Game Modes:** Two distinct modes: Simple TCR and Enhanced TCR.

**Game Elements (Applicable to both modes unless specified):**

*   **Players:** Two players per game session.
*   **Towers:** Each player has 1 King Tower and 2 Guard Towers with stats (HP, ATK, DEF, CRIT%, EXP Yield) defined in `config/towers.json`.
*   **Troops:** Players can deploy troops from a defined list (Pawn, Bishop, Rook, Knight, Prince, Queen) with stats (HP, ATK, DEF, Mana Cost) defined in `config/troops.json`.
*   **Targeting Rule:** Guard Tower 1 must be destroyed before Guard Tower 2 or the King Tower can be targeted by troops.
*   **Combat:**
    *   Damage Formula: `DMG = Attacker_ATK - Defender_DEF`. If DMG < 0, DMG = 0. `Defender_HP = Defender_HP - DMG`.
    *   Troops only attack Towers.
    *   Towers attack Troops (see specific mode rules below).
*   **Queen Troop:** Costs 5 Mana. Deployment is a one-time action healing the friendly tower with the lowest absolute HP by 300 (up to max HP). Does not persist on the board. Consumes turn in Simple mode.
*   **EXP & Leveling:**
    *   Players gain EXP by destroying enemy towers (amount specified per tower type) and by winning (30 EXP) or drawing (10 EXP) a match (Enhanced mode only for Win/Draw EXP).
    *   EXP required for the next level increases cumulatively by 10% per level, starting with 100 EXP needed for Level 2. (Level N+1 needs `100 * (1.1 ^ (N-1))` total EXP from the previous level).
    *   Each player level increases *all* their Troops' and Towers' base HP, ATK, and DEF by 10% cumulatively per level. (Level N Stat = BaseStat \* (1.1 ^ (N-1))).

**Simple TCR Rules:**

*   **Turns:** Players take turns deploying one troop.
*   **Troop Deployment:** Player chooses one troop from the available list to deploy.
*   **Troop Attack:** Deployed troops attack the lowest HP valid enemy tower on the player's turn *after* the turn they were deployed. If a troop survives combat, it remains and attacks again on subsequent turns.
*   **Tower Attack:** If a tower was attacked by an enemy troop during the opponent's turn, the tower attacks back once at the *end* of that opponent's turn, targeting the last troop that attacked it (or oldest if multiple/last is gone).
*   **Win Condition:** The first player to destroy the opponent's King Tower wins immediately.

**Enhanced TCR Rules:**

*   **Real-Time:** No turns. Game runs continuously for 3 minutes.
*   **MANA System:**
    *   Players start with 5 MANA.
    *   MANA regenerates at 1 MANA per second.
    *   Maximum MANA is 10.
    *   Deploying troops costs MANA as specified in `config/troops.json`. Players cannot deploy if they lack sufficient MANA.
*   **Continuous Attack:**
    *   Deployed troops attack automatically **once per second**. They target the opponent's lowest absolute HP valid tower (respecting GT1 rule).
    *   Towers attack automatically **once per second** if a valid enemy troop target exists (last attacker or oldest attacker).
*   **CRIT Chance:**
    *   Attacks from Towers and Troops have a chance to be critical hits (%). CRIT chance is defined per Tower/Troop type in the config files (Note: Appendix only listed CRIT for Towers - **Assumption:** Troops have 0% base CRIT unless specified otherwise, or we need base CRIT values for troops too. **Let's assume 0% base CRIT for all troops for simplicity**).
    *   Critical Hit Damage: `DMG = (Attacker_ATK * 1.2) - Defender_DEF`.
*   **Win Conditions:**
    *   Instant Win: First player to destroy the opponent's King Tower.
    *   Timeout: If 3 minutes elapse, the player who destroyed more towers wins.
    *   Draw: If time expires and both players destroyed the same number of towers.
*   **EXP Awards:** Win: 30 EXP, Draw: 10 EXP (awarded at game end, in addition to EXP from destroyed towers).

## Appendix

### Tower Stats

| Type        | HP   | ATK | DEF | CRIT | EXP |
|-------------|------|-----|-----|------|-----|
| King Tower  | 2000 | 500 | 300 | 10%  | 200 |
| Guard Tower | 1000 | 300 | 100 | 5%   | 100 |

### Troop Stats

| Name   | HP   | ATK | DEF | MANA | EXP | Special                                      |
|--------|------|-----|-----|------|-----|----------------------------------------------|
| Pawn   | 50   | 150 | 100 | 3    | 5   |                                              |
| Bishop | 100  | 200 | 150 | 4    | 10  |                                              |
| Rook   | 250  | 200 | 200 | 5    | 25  |                                              |
| Knight | 200  | 300 | 150 | 5    | 25  |                                              |
| Prince | 500  | 400 | 300 | 6    | 50  |                                              |
| Queen  | N/A  | N/A | N/A | 5    | 30  | Heals the friendly tower with lowest HP by 300 |

**Phase 2: Proposed Architecture & Design**

We will use a standard Client-Server architecture.

1.  **Server:**
    *   **Responsibilities:**
        *   Listens for incoming client connections (TCP).
        *   Handles client authentication (username/password against stored data).
        *   Manages player matchmaking (pairing authenticated users).
        *   Creates and manages game sessions (one per pair of players).
        *   Enforces game rules (Simple or Enhanced mode).
        *   Runs the game loop (turn-based or real-time timer).
        *   Calculates combat, manages HP, MANA, EXP, levels.
        *   Synchronizes game state changes to connected clients.
        *   Persists player data (EXP, levels) and loads game configuration (troop/tower specs) from JSON files.
    *   **Concurrency:** Use Goroutines for handling individual client connections and separate game sessions. Use Channels for communication between connection handlers and game logic, ensuring thread safety.

2.  **Client:**
    *   **Responsibilities:**
        *   Connects to the server (TCP).
        *   Prompts user for username/password and sends credentials.
        *   Receives and displays game state information from the server (tower HP, troop status, mana, timer, opponent actions, game results).
        *   Prompts user for actions (e.g., "deploy troop X", potentially "target Y" if needed).
        *   Sends user actions to the server.
        *   Handles server messages (authentication success/failure, game start, state updates, game end).
    *   **Implementation:** Simple command-line interface (CLI).

3.  **Communication Protocol:**
    *   **Transport:** TCP.
    *   **Serialization:** JSON. Define clear message structures for client-to-server (C2S) and server-to-client (S2C) communication.
        *   *Example C2S Messages:* `{"type": "login", "payload": {"username": "user1", "password": "pwd"}}`, `{"type": "deploy_troop", "payload": {"troop_id": "troop_A", "target_tower_id": "opponent_guard1"}}` (Target might be implicit based on rules).
        *   *Example S2C Messages:* `{"type": "auth_result", "payload": {"success": true}}`, `{"type": "game_start", "payload": {"opponent_username": "user2", "initial_state": {...}}}` , `{"type": "state_update", "payload": {"towers": [...], "troops": [...], "mana": 5, "time_left": 180}}`, `{"type": "game_event", "payload": {"message": "Player1's Knight destroyed Player2's Guard Tower 1!"}}`, `{"type": "game_over", "payload": {"winner": "user1", "reason": "King Tower destroyed", "exp_earned": 30}}`.

4.  **Data Structures (Core Models):**
    *   `Player`: `ID`, `Username`, `HashedPassword`, `EXP`, `Level`, `Connection` (transient `net.Conn`), `GameID` (transient), `CurrentMana` (Enhanced), `Towers` (map/slice), `AvailableTroops` (map/slice based on clarification), `ActiveTroops` (map/slice).
    *   `Tower`: `ID`, `Name`, `BaseHP`, `BaseATK`, `BaseDEF`, `CurrentHP`, `OwnerPlayerID`.
    *   `TroopSpec`: `ID`, `Name`, `BaseHP`, `BaseATK`, `BaseDEF`, `ManaCost` (Enhanced), `BaseCritChance` (Enhanced). (Loaded from JSON).
    *   `ActiveTroop`: `InstanceID`, `SpecID`, `CurrentHP`, `OwnerPlayerID`, `TargetID` (transient).
    *   `Game`: `ID`, `Players` [2]*Player, `GameState` (e.g., `Waiting`, `RunningSimple`, `RunningEnhanced`, `Finished`), `CurrentTurnPlayerID` (Simple), `StartTime` (Enhanced), `EndTime` (Enhanced), `BoardState` (containing all active troops and tower statuses).
    *   `PlayerData`: `Username`, `HashedPassword`, `EXP`, `Level`. (For JSON persistence).
    *   `GameConfig`: Contains maps/slices of `TowerSpec` and `TroopSpec`. (Loaded from JSON).

5.  **Persistence:**
    *   Use Go's `encoding/json` package.
    *   Player data: Store one JSON file per player (e.g., `data/players/username.json`) or a single JSON file containing a map of all players. A single file is simpler initially, but separate files scale better if many players were expected (though not likely critical here).
    *   Game config: Store troop and tower base stats in separate JSON files (e.g., `config/troops.json`, `config/towers.json`).

**Phase 3: Project Structure (Go Modules)**

```
net-centric-project (text-clash-royale)/
├── cmd/
│   ├── tcr-server/
│   │   └── main.go       # Server executable entry point
│   └── tcr-client/
│       └── main.go       # Client executable entry point
├── internal/
│   ├── server/
│   │   ├── server.go     # Main server logic (listening, connection handling)
│   │   ├── auth.go       # Authentication logic
│   │   ├── matchmaking.go # Pairing players
│   │   └── session.go    # Game session management
│   ├── client/
│   │   ├── client.go     # Main client logic (connection, input/output loop)
│   │   └── handler.go    # Handling server messages
│   ├── game/
│   │   ├── game.go       # Core game state structure and management
│   │   ├── logic_simple.go # Logic specific to Simple TCR rules
│   │   ├── logic_enhanced.go # Logic specific to Enhanced TCR rules
│   │   ├── combat.go     # Damage calculation, CRIT logic
│   │   ├── progression.go # EXP and Leveling logic
│   │   └── models.go     # Game-related data structures (Tower, Troop, PlayerInGame)
│   ├── models/
│   │   ├── player.go     # Player data structure (for persistence)
│   │   └── config.go     # Structures for loading troop/tower specs
│   ├── network/
│   │   ├── protocol.go   # Defines C2S and S2C message structures
│   │   └── codec.go      # Helper functions for encoding/decoding JSON messages over TCP
│   └── persistence/
│       └── storage.go    # Functions for loading/saving JSON data (player profiles, config)
├── pkg/          # Optional: Shared libraries if needed (e.g., custom logger)
│   └── logger/
│       └── logger.go
├── data/         # Default directory for persistent player data
│   └── players/
├── config/       # Default directory for game configuration files
│   ├── troops.json
│   └── towers.json
├── go.mod
├── go.sum
└── README.md
```

**Phase 4: Development Plan (Iterative)**

*   **Sprint 0: Setup & Foundation (Requires Appendix Clarification First!)**
    * [X]   Finalize requirements based on clarifications.
    * [X]   Set up Go module project structure.
    * [X]   Define core data structures (`models`, `game.models`).
    * [X]   Define basic network `protocol` message types (JSON).
    * [X]   Implement basic TCP server (`cmd/tcr-server`, `internal/server`) capable of accepting connections.
    * [X]   Implement basic TCP client (`cmd/tcr-client`, `internal/client`) capable of connecting.
    * [X]   Implement basic `network/codec` for sending/receiving simple JSON messages.
    * [X]   Implement `persistence/storage` to load dummy `config/towers.json` and `config/troops.json`.

*   **Sprint 1: Simple TCR - Connection & Game Setup**
    * [X]   Implement basic user authentication (`internal/server/auth`) - store hashed passwords in memory or basic files initially (`internal/persistence`).
    * [X]   Implement client-side login prompt and message sending.
    * [X]   Implement basic matchmaking (`internal/server/matchmaking`) - pair the first two authenticated users.
    * [X]   Implement game session creation (`internal/server/session`).
    * [X]   Implement sending initial game state (Simple TCR rules) to both clients upon match start.
    * [X]   Implement client-side display of the initial game board/state.

*   **Sprint 2: Simple TCR - Core Gameplay Loop**
    * [X]   Implement turn-based logic in `internal/game/logic_simple.go`.
    * [X]   Implement client command for deploying a troop (based on clarified rules).
    * [X]   Implement server-side handling of the deploy command.
    * [X]   Implement combat logic (`internal/game/combat.go`) including the Simple damage formula and targeting rules (Guard1 first).
    * [X]   Implement troop attack sequence (including "continue attacking" if a tower is destroyed - based on clarification).
    * [X]   Implement sending state updates (HP changes, troop/tower destruction, turn change) to clients.
    * [X]   Update client display to reflect game state changes.
    * [u]   Implement win condition checking for Simple TCR.
    * [u]   Implement game end message and basic session cleanup.

*   **Sprint 3: Persistence & Refinement**
    * [X]   Implement robust JSON loading/saving for player data (`PlayerData` including hashed passwords) using `internal/persistence`.
    * []   Ensure troop/tower specs are loaded correctly from JSON `config` files (using the actual Appendix data).
    * []   Add basic error handling and logging (`pkg/logger` or standard `log` package).
    * []   Refine client text UI for better clarity.
    * []   Test Simple TCR thoroughly.

*   **Sprint 4: Enhanced TCR - Real-time & Core Mechanics**
    * []   Adapt the server game loop (`internal/game/logic_enhanced.go`) for real-time (3-minute timer).
    * []   Implement MANA system (cost for troop deployment, regeneration). Update `PlayerData` or `PlayerInGame`.
    * []   Update client command for deployment to check MANA.
    * []   Implement continuous attack logic (based on clarification - e.g., attacks every X seconds). Goroutines within the game session might manage troop actions.
    * []   Implement CRIT chance (`internal/game/combat.go`).
    * []   Implement Enhanced TCR targeting rules (Guard1 first).
    * []   Implement Enhanced TCR win conditions (King Tower destruction or timeout).
    * []   Update client to display MANA, timer, and handle more frequent state updates.

*   **Sprint 5: Enhanced TCR - Progression & Persistence**
    * []   Implement EXP awarding system (`internal/game/progression.go`).
    * []   Implement Leveling system (calculating required EXP, applying stat boosts - based on clarification).
    * []   Update `PlayerData` persistence to store/load EXP and Level.
    * []   Modify combat and game logic to use leveled stats for troops/towers.
    * []   Test Enhanced TCR thoroughly.

*   **Sprint 6: Final Testing, Documentation & Demo Prep**
    * []   Conduct integration testing with multiple clients.
    * []   Test edge cases (disconnects, invalid input).
    * []   Write `README.md` explaining how to build and run the server and client.
    * []   Prepare for the demonstration, ensuring both Simple and Enhanced modes work as required.
    * []   Code cleanup and final review.

**Phase 5: Testing Strategy**

*   **Unit Tests:** Test individual functions, especially in `game/combat`, `game/progression`, `persistence`, and `network/codec`.
*   **Integration Tests:** Test interactions between components (e.g., client connecting, authenticating, server creating a game session).
*   **End-to-End Tests:** Run the server and multiple client instances manually or via scripts to simulate full game sessions for both Simple and Enhanced modes. Test win/loss/draw conditions, EXP gain, leveling, mana usage, etc.

**Phase 6: Deliverables Checklist**

*    Source Code (well-structured Go project).
*    `README.md` with build/run instructions.
*    JSON configuration files for troops/towers.
*    JSON files for persistent player data (will be created/updated during runtime).
*    Live demonstration of the application (both Simple and Enhanced modes).

