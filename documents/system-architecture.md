
**System Architecture Document: Text-Based Clash Royale (TCR)**

**1. Introduction**

This document outlines the system architecture for the Text-Based Clash Royale (TCR) application. TCR is a two-player, client-server strategy game implemented in Go. It provides a simplified, text-based experience inspired by Clash Royale, featuring two distinct game modes: Simple (turn-based) and Enhanced (real-time with additional mechanics). The system facilitates network communication, game logic execution, state management, and data persistence.

**2. Goals and Requirements Summary**

The architecture aims to fulfill the following key requirements derived from the project specification and subsequent clarifications:

*   **Client-Server Model:** The system must operate with a central server managing game state and multiple clients connecting for gameplay.
*   **Network Communication:** Communication between clients and the server must use the TCP protocol for reliable message delivery. Data exchange will utilize JSON format.
*   **User Authentication:** Players must authenticate using a username and password. Passwords must be securely hashed using bcrypt on the server side before persistence.
*   **Dual Game Modes:** Support two modes:
    *   **Simple TCR:** Turn-based, basic combat, sequential tower destruction (GT1 -> GT2/King).
    *   **Enhanced TCR:** Real-time (3-min limit), Mana system, CRIT chance, continuous attacks, EXP/Leveling system, complex win conditions (King destruction or tower count).
*   **Game Logic:** Implement core game mechanics including:
    *   Troop deployment (from a shared list, costing Mana in Enhanced).
    *   Tower defense and troop attacks (troops attack towers only).
    *   Specific targeting rules (Guard Tower 1 must be destroyed first).
    *   Combat resolution using `DMG = ATK - DEF` (with CRIT modifier in Enhanced).
    *   Queen troop's unique healing ability (one-time effect).
*   **State Management:** The server must manage game state including tower/troop HP, player Mana (Enhanced), game timer (Enhanced), current turn (Simple), and active game sessions.
*   **Progression System (Enhanced):** Implement EXP gain (tower destruction, win/draw) and a Leveling system affecting player Tower/Troop stats cumulatively (10% per level).
*   **Persistence:**
    *   Player data (Username, Hashed Password, EXP, Level) must be saved to and loaded from JSON files.
    *   Base Tower and Troop stats (HP, ATK, DEF, CRIT% for Towers, Mana Cost for Troops) must be loaded from configuration JSON files.

**3. Architectural Style**

A **Client-Server** architectural style is employed.

*   **Server:** Acts as the central authority, managing game instances, enforcing rules, calculating outcomes, synchronizing state, and handling persistence.
*   **Client:** Acts as the presentation and input layer for the player, displaying game information received from the server and sending player actions back to the server.

This style is chosen for its suitability in managing shared game state, centralizing game logic, and facilitating interaction between multiple players.

**4. Component Breakdown**

The system is composed of the following major components:

*   **TCR Server:** The backend application responsible for overall game orchestration.
    *   **Listener:** Accepts incoming TCP connections from clients.
    *   **Authentication Manager:** Verifies client credentials against persisted player data (using bcrypt).
    *   **Matchmaker:** Pairs authenticated clients waiting for a game.
    *   **Session Manager:** Creates, manages, and terminates active game sessions (one per pair of players).
    *   **Game Logic Engine:** Contains the core rules and state machines for both Simple and Enhanced TCR modes. Calculates combat, handles targeting, manages turns/timers, applies status effects (heal), and determines win/loss/draw conditions. Includes sub-components for:
        *   Simple Mode Logic
        *   Enhanced Mode Logic (Timer, Mana, CRIT)
        *   Combat Calculation
        *   Progression Calculation (EXP/Leveling)
    *   **State Synchronizer:** Broadcasts relevant game state changes to the clients involved in a session.
    *   **Persistence Interface:** Handles reading configuration files and reading/writing player data JSON files.
*   **TCR Client:** The frontend application used by players.
    *   **Network Client:** Establishes and maintains the TCP connection to the server. Encodes/decodes JSON messages.
    *   **User Interface (CLI):** Renders game state information (board, HP, Mana, timer, messages) to the console.
    *   **Input Handler:** Captures player commands (login, deploy troop) from the console.
    *   **Message Processor:** Interprets messages received from the server (auth results, state updates, game events, game over) and updates the UI accordingly.
*   **Network Layer:**
    *   **Protocol:** TCP/IP ensures reliable, ordered delivery of messages between Client and Server.
    *   **Data Format:** JSON provides a structured, human-readable format for messages.
*   **Persistence Layer:**
    *   **Storage:** Uses the local filesystem.
    *   **Player Data:** Individual JSON files per player (e.g., `data/players/username.json`) storing `Username`, `HashedPassword`, `EXP`, `Level`.
    *   **Game Configuration:** JSON files (e.g., `config/towers.json`, `config/troops.json`) storing base stats and parameters for game entities.

**5. Data Model (Key Structures)**

*   **`PlayerAccount` (Persistence):** `Username`, `HashedPassword`, `EXP`, `Level`.
*   **`GameConfig` (Loaded):** Contains slices/maps of `TowerSpec` and `TroopSpec` structs holding base stats (HP, ATK, DEF, CRIT%[Tower], ManaCost[Troop], EXPYield[Tower]).
*   **`GameSession` (Runtime):** `SessionID`, `Players` [2]*`PlayerInGame`, `Mode` (Simple/Enhanced), `GameState` (Waiting, Running, Finished), `BoardState` (containing `TowerInstance`s and `ActiveTroop`s), `CurrentTurnPlayerID` (Simple), `GameTimer` (Enhanced), etc.
*   **`PlayerInGame` (Runtime):** Reference to `PlayerAccount`, `Connection` (`net.Conn`), `CurrentMana` (Enhanced), references to their `TowerInstance`s and `ActiveTroop`s.
*   **`TowerInstance` (Runtime):** `SpecID`, `OwnerPlayerID`, `CurrentHP`, `MaxHP` (calculated with level), `CurrentATK`/`DEF` (calculated with level).
*   **`ActiveTroop` (Runtime):** `InstanceID`, `SpecID`, `OwnerPlayerID`, `CurrentHP`, `MaxHP` (calculated), `CurrentATK`/`DEF` (calculated), `TargetID`.
*   **`NetworkMessage` (Communication):** Standard structure like `{"type": "message_type", "payload": {...}}` used for all C2S and S2C communication.

**6. Communication Protocol**

*   **Transport:** TCP.
*   **Serialization:** JSON.
*   **Key Message Types:**
    *   **C2S (Client-to-Server):** `LoginRequest`, `DeployTroopCommand`, `GetGameStateRequest` (optional), `QuitMatch`.
    *   **S2C (Server-to-Client):** `LoginResponse` (Success/Failure), `MatchFound`, `GameStart`, `GameStateUpdate` (periodic or event-driven, includes HP, Mana, Timer, Deployed Troops, Tower Status), `GameEvent` (e.g., "Tower Destroyed", "Troop Deployed", "Heal Applied"), `TurnNotification` (Simple), `GameOver` (Winner/Loser/Draw, Reason, EXP Awarded).

**7. Concurrency Model (Server)**

The server will leverage Go's concurrency features:

*   **Connection Handling:** A dedicated Goroutine will be spawned for each connected client to handle reading requests and writing responses independently.
*   **Game Session Management:** Each active game session (a match between two players) will run in its own Goroutine. This Goroutine manages the game loop (turn progression for Simple, timer and real-time updates for Enhanced) and game state.
*   **Communication:** Go Channels will be used for safe communication between client connection Goroutines and their corresponding game session Goroutine. For example:
    *   A client Goroutine receives a `DeployTroopCommand` and sends it over a channel to the game session Goroutine.
    *   The game session Goroutine processes game logic and broadcasts `GameStateUpdate` messages back to the relevant client Goroutines via channels.

**8. Deployment View**

The system consists of two main executable artifacts:

*   **`tcr-server`:** A single instance runs on a machine, listening on a configured TCP port.
*   **`tcr-client`:** Multiple instances can be run by players on their machines, configured to connect to the server's IP address and port.

Data files (player profiles, configuration) reside on the server machine's filesystem in designated directories (`data/`, `config/`).

**9. Technology Stack**

*   **Programming Language:** Go (Golang)
*   **Networking:** Go standard library `net` package (TCP Sockets).
*   **Serialization:** Go standard library `encoding/json`.
*   **Password Hashing:** `golang.org/x/crypto/bcrypt`.
*   **Concurrency:** Go Goroutines and Channels.

