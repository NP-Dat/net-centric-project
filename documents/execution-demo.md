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
4. The client will attempt to connect to the server. Once connected, you will see a prompt for interaction.

## Client Commands and Interactions

(handle in `cmd\tcr-client\main.go` , remember to update it)

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

### 3. Send a Message
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

### 4. Quit
- Command: `quit` or `exit`
- Description: Disconnects the client from the server.
- Steps:
  1. Type `quit` or `exit` to disconnect.
- Example:
  ```
  > quit
  Disconnecting from server...
  ```

### 5. Help
- Command: `help`
- Description: Displays a list of available commands.
- Example:
  ```
  > help
  Available commands:
    login - Login to the server interactively
    join - Join the matchmaking queue
    send <message> - Send a message to the server
    quit/exit - Disconnect and quit
    help - Show this help message
  ```

## Notes
- Ensure the server is running before starting the client.
- If the client disconnects unexpectedly, restart the client and log in again.
- Matchmaking requires at least two clients to join the queue.