package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/StephenBirch/message-delivery-system/client"
	"github.com/gorilla/websocket"
)

var (
	helpText = "\nSelect a number from:\n1: Identify\n2: List users\n3: Relay message from stdin\n4: Relay message from file\n5: Exit\n"
)

func main() {
	address := flag.String("address", "localhost:8080", "The address&port of the hub")
	flag.Parse()

	c, err := client.New(*address)
	if err != nil {
		log.Fatal(err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/ws?id=%d", c.Address, c.ID), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	// go func for sending messages
	go func() {
		for {
			select {
			case msg := <-c.Sending:
				b, err := json.Marshal(msg)
				if err != nil {
					fmt.Printf("Failed to Marshal msg: %v\n", err)
					return
				}

				err = conn.WriteMessage(websocket.TextMessage, b)
				if err != nil {
					fmt.Println("write:", err)
					return
				}
			}
		}
	}()

	// go func for receiving messages
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Failed to read message from websocket: %v\nExiting.", err)
				os.Exit(1)
			}
			fmt.Printf("Incoming data: %s\n", message)
		}
	}()

	fmt.Printf("\nConnected to hub %s. Your ID: %d\n", *address, c.ID)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Println(helpText)
		scanner.Scan()
		switch scanner.Text() {
		// Identify
		case "1":
			id, err := c.Identify()
			if err != nil {
				fmt.Printf("Failed to identify self: %v", err)
				continue
			}
			fmt.Println("Your ID:", id)
		// List Users
		case "2":
			ids, err := c.ListUsers()
			if err != nil {
				fmt.Printf("Failed to get list of users: %v\n", err)
				continue
			}
			fmt.Printf("Connected users: %v\n", ids)
		// Relay message from stdin
		case "3":
			fmt.Printf("Enter the recipients IDs (CSV)\n> ")
			scanner.Scan()

			recipients := scanner.Text()
			recipients = strings.TrimSpace(recipients)

			// Check we're not sending to more recipients than maxRecipients
			if err := client.VerifyRecipients(recipients); err != nil {
				fmt.Printf("Invalid recipients: %s\n", err)
				continue
			}

			fmt.Printf("Enter data to send\n> ")
			scanner.Scan()

			// If they somehow type out a insanely large message
			if len(scanner.Bytes()) > int(client.MaxDataSize) {
				fmt.Printf("Data is larger than max size(%d) was %d", client.MaxDataSize, len(scanner.Bytes()))
				continue
			}

			c.Sending <- client.SendingMessage{Recipients: recipients, Data: scanner.Bytes()}
			continue
		// Relay message from file
		case "4":
			fmt.Printf("Enter the recipients IDs (CSV)\n> ")
			scanner.Scan()

			recipients := scanner.Text()
			recipients = strings.TrimSpace(recipients)

			if err := client.VerifyRecipients(recipients); err != nil {
				fmt.Printf("Invalid recipients: %s\n", err)
				continue
			}

			fmt.Printf("Enter filepath of data to send\n> ")
			scanner.Scan()

			if err := client.VerifyFile(scanner.Text()); err != nil {
				fmt.Printf("Invalid file: %s\n", err)
				continue
			}

			// If it's under max size continue to read & send message to recipients
			b, err := ioutil.ReadFile(scanner.Text())
			if err != nil {
				fmt.Printf("Failed to open file: %s\n", err)
				continue
			}

			c.Sending <- client.SendingMessage{Recipients: recipients, Data: b}
			continue
		// Exit
		case "5":
			conn.Close()
			fmt.Printf("Goodbye")
			os.Exit(0)
		default:
			fmt.Println(helpText)
			continue
		}
	}
}
