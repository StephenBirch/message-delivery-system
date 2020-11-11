package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/StephenBirch/message-delivery-system/client"
	"github.com/gorilla/websocket"
)

var (
	helpText = "Please select a number from:\n1: Identify\n2: List users\n3: Relay message from stdin\n4: Relay message from file\n5: Exit\n"
)

func main() {
	address := flag.String("address", "localhost:8080", "The address&port of the hub")
	flag.Parse()

	c, err := client.New(*address)
	if err != nil {
		log.Fatal(err)
	}

	// go func for sending/receiving messages
	go func() {
		conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/ws?id=%d", c.Address, c.ID), nil)
		if err != nil {
			log.Fatal("dial:", err)
		}
		defer conn.Close()
		t := time.NewTicker(time.Second * 5)
		for {
			select {
			case msg := <-c.Sending:
				b, err := json.Marshal(msg)
				if err != nil {
					log.Printf("Failed to Marshal msg: %v", err)
					return
				}

				err = conn.WriteMessage(websocket.TextMessage, b)
				if err != nil {
					log.Println("write:", err)
					return
				}
			case <-t.C:
				_, message, err := conn.ReadMessage()
				if err != nil {
					log.Println("read:", err)
					return
				}
				log.Printf("Received: %s", message)
			}
		}
	}()

	fmt.Printf("\nConnected to hub %s. Your ID: %d\n\n", *address, c.ID)

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println(helpText)
	for scanner.Scan() {
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
				fmt.Printf("Failed to get list of users: %v", err)
				continue
			}
			fmt.Printf("Connected users: %v\n", ids)
		// Relay message from stdin
		case "3":
			fmt.Println("Enter the recipients IDs (CSV)")
			scanner.Scan()
			recipients := scanner.Text()
			fmt.Println("Enter data to send")
			c.Sending <- client.SendingMessage{Recipients: recipients, Data: scanner.Bytes()}
			continue
		// Relay message from file
		case "4":
			fmt.Println("NYI")
			continue
		// Exit
		case "5":
			os.Exit(0)
		default:
			fmt.Println(helpText)
			continue
		}

	}
}
