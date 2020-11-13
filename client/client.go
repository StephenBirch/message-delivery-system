package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/StephenBirch/message-delivery-system/types"
	"github.com/gorilla/websocket"
)

var (
	MaxRecipients = 255
	MaxDataSize   = int64(1024000) // 1024 kilobyes
)

type Client struct {
	ID      uint64
	Address string
	Sending chan types.SendingMessage
}

func New(address string) (*Client, error) {
	client := &Client{
		Address: address,
		Sending: make(chan types.SendingMessage),
	}

	id, err := client.Register()
	if err != nil {
		return nil, fmt.Errorf("failed to register client: %v", err)
	}

	client.ID = id

	return client, nil
}

func (c *Client) do(address string, object interface{}) error {
	resp, err := http.Get(address)
	if err != nil {
		return fmt.Errorf("failed to reach hub %s: %s", c.Address, err)
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response from %s: %s", c.Address, err)
	}

	if err := json.Unmarshal(b, &object); err != nil {
		return fmt.Errorf("failed to unmarshal response from %s: %s", c.Address, err)
	}
	return nil

}

// Register is used to get an ID, and is automatically called by New()
func (c *Client) Register() (uint64, error) {
	var id uint64
	return id, c.do(fmt.Sprintf("http://%s/register", c.Address), &id)
}

// ListUsers is used to wrap the /users endpoint from the hub
func (c *Client) ListUsers() (types.ListResponse, error) {
	var resp types.ListResponse
	return resp, c.do(fmt.Sprintf("http://%s/users?id=%d", c.Address, c.ID), &resp)
}

// Identify is used to wrap the /identify endpoint, using the client.ID to obtain it back after checking with the hub
func (c *Client) Identify() (uint64, error) {
	var id uint64
	return id, c.do(fmt.Sprintf("http://%s/identify?id=%d", c.Address, c.ID), &id)
}

// VerifyRecipients checks that there's not more than MaxRecipient entries, and that they can all be parsed as uint64
func VerifyRecipients(recipients string) error {
	ids := strings.Split(recipients, ",")
	if len(ids) > MaxRecipients {
		return fmt.Errorf("recipients exceed max length(%d) was: %d", MaxRecipients, len(ids))
	}

	for _, id := range ids {
		_, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			return fmt.Errorf("recipient %s could not be parsed as uint64: %s", id, err)
		}
	}
	return nil
}

// VerifyFile checks that the file exists, and that it is smaller than MaxDataSize
func VerifyFile(filepath string) error {
	f, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open file: %s", err)
	}
	stats, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %s", err)
	}
	if stats.Size() > MaxDataSize {
		return fmt.Errorf("file exceeded max size(%d) was: %d", MaxDataSize, stats.Size())
	}

	return nil
}

// InitWebsocket is a one time call to upgrade the connection to a websocket for sending/receiving messages
func (c *Client) InitWebsocket() (*websocket.Conn, error) {
	conn, resp, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/ws?id=%d", c.Address, c.ID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to dial websocket: %s", err)
	}
	// 101 = Switching Protocols, expected for Upgrade requests
	if resp.StatusCode != 101 {
		return nil, fmt.Errorf("Non-101 return code: %d", resp.StatusCode)
	}
	return conn, nil
}

// WriteMessages is a blocking call constantly writing messages from the clients channel
func (c *Client) WriteMessages(conn *websocket.Conn) error {
	if conn == nil {
		return fmt.Errorf("conn can't be nil")
	}
	for {
		select {
		case msg := <-c.Sending:
			b, err := json.Marshal(msg)
			if err != nil {
				return fmt.Errorf("failed to Marshal message: %s", err)
			}

			err = conn.WriteMessage(websocket.TextMessage, b)
			if err != nil {
				return fmt.Errorf("failed to write message: %s", err)
			}
		}
	}
}

// ReadMessages is a blocking call constantly checking for messages from the websocket connection and writing them out to stdout
func (c *Client) ReadMessages(conn *websocket.Conn) error {
	if conn == nil {
		return fmt.Errorf("conn can't be nil")
	}
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("failed to read message: %v", err)
		}
		fmt.Printf("Incoming data: %s\n", message)
	}
}
