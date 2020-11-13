# message-delivery-system

This project is an excercise to develop a simple message delivery system using websockets.

## How to run

1. Start the hub with: `go run cmd/hub/main.go --port=<port>` you can exclude port to run on 8080
2. Start one or more clients with: `go run cmd/client/main.go --address=<IP>:<port>` you can exclude address to run on localhost

## Functions

Once running both a hub and a client, you're able to perform the following functions:

1. Identify self
2. List users currently connected to the hub
3. Relay a message from stdin
4. Relay a message from file
5. Exit

