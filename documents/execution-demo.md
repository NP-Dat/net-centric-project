# Execution Demo Instructions

This document provides instructions on how to execute the server and client, as well as how the client can interact with the server.

## Running the Server

1. Open a terminal.
2. Navigate to the server directory:
   ```bash
   cd "d:\Phuc Dat\IU\MY PROJECT\Golang\net-centric-project\cmd\tcr-server"
   ```
3. Run the server:
   ```bash
   go run main.go
   ```
   
   You can also specify a custom log level:
   ```bash
   go run main.go --logLevel=debug
   ```
4. The server will start and listen for incoming client connections. You should see a message indicating the server is running, e.g., `Server started on localhost:8080`.

## Running the Client

1. Open another terminal.
2. Navigate to the client directory:
   ```bash
   cd "d:\Phuc Dat\IU\MY PROJECT\Golang\net-centric-project\cmd\tcr-client"
   ```
3. Run the client:
   ```bash
   go run main.go
   ```
   
   You can also specify a custom log level:
   ```bash
   go run main.go --logLevel=debug
   ```
4. The client will attempt to connect to the server. Once connected, you will see a prompt for interaction.

## Client Commands and Interactions

(handle in `cmd\tcr-client\main.go` and `internal\client\handler.go`, remember to update it)

After connecting to the server, the client can execute the following commands:

### 1. Login
- Command: `login`
- Description: Prompts the user to log in interactively.
- Steps:
  1. Enter your username and password when prompted.
  2. If the username is new, an account will be created.
  3. If the username exists, the password will be verified.
- Example:
  ```
  === Text Clash Royale Login ===
  Username: player1
  Password: secret
  Authentication successful. Welcome, player1!
  ```

### 2. Join Matchmaking Queue
- Command: `join`
- Description: Adds the client to the matchmaking queue.
- Steps:
  1. Ensure you are logged in.
  2. Type `join` to enter the matchmaking queue.
  3. Wait for another player to join the queue.
- Example:
  ```
  > join
  You have been added to the matchmaking queue. Waiting for opponent...
  ```

### 3. Deploy a Troop
- Command: `deploy <troop_id>`
- Description: Deploys a troop during an active game.
- Steps:
  1. Ensure you are logged in and in an active game.
  2. Type `deploy` followed by the troop type.
  3. Available troops: pawn, bishop, rook, knight, prince, queen
- Example:
  ```
  > deploy knight
  ⚔️ Deploying Knight...
  ```
- Note: You can only deploy troops during your turn in Simple mode, or if you have enough mana in Enhanced mode.

### 4. Send a Message
- Command: `send <message>`
- Description: Sends a chat message to all connected clients.
- Steps:
  1. Ensure you are logged in.
  2. Type `send` followed by your message.
- Example:
  ```
  > send Hello, everyone!
  Message sent
  ```

### 5. Quit
- Command: `quit` or `exit`
- Description: Disconnects the client from the server.
- Steps:
  1. Type `quit` or `exit` to disconnect.
- Example:
  ```
  > quit
  Disconnecting from server...
  ```

### 6. Help
- Command: `help`
- Description: Displays a list of available commands.
- Example:
  ```
  > help
  Available commands:
    login <username> <password> - Log in to the server
    join - Join the matchmaking queue
    deploy <troop> - Deploy a troop in the current game
    quit - Disconnect from the server
    help - Display this help message
  ```

### 7. Change Log Level (Debug Command)
- Command: `debug loglevel <level>`
- Description: Changes the logging verbosity level at runtime.
- Available levels: debug, info, warn, error
- Steps:
  1. Type `debug loglevel` followed by the desired log level.
- Example:
  ```
  > debug loglevel debug
  Log level set to DEBUG
  ```
- Note: This command is useful for troubleshooting. Higher verbosity (debug) shows more details, while lower verbosity (error) shows only critical issues.

## Log Levels

When starting the client or server, you can specify the desired log level:

| Log Level | Description                                              |
|-----------|----------------------------------------------------------|
| debug     | Most verbose. Shows all messages including detailed info. |
| info      | Standard level. Shows normal operational messages.        |
| warn      | Shows warnings and more severe issues only.              |
| error     | Shows only error messages and critical failures.         |

Example of setting log level when starting the client:
```bash
go run main.go --logLevel=debug
```

## Notes
- Ensure the server is running before starting the client.
- If the client disconnects unexpectedly, restart the client and log in again.
- Matchmaking requires at least two clients to join the queue.
- `ctrl + C` for in Server terminal to close the Server.

