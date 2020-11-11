package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

type ListResponse struct {
	IDs []uint64
}

type Client struct {
	ID      uint64
	Address string
	Sending chan SendingMessage
}

type SendingMessage struct {
	Recipients string
	Data       []byte
}

func New(address string) (*Client, error) {
	client := &Client{
		Address: address,
		Sending: make(chan SendingMessage),
	}

	id, err := client.Register()
	if err != nil {
		return nil, fmt.Errorf("failed to register client: %v", err)
	}

	client.ID = id

	return client, nil
}

func (c *Client) Register() (uint64, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/register", c.Address))
	if err != nil {
		return 0, fmt.Errorf("failed to reach hub %s: %s", c.Address, err)
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response from %s: %s", c.Address, err)
	}

	parsedID, err := strconv.ParseUint(string(b), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to decode userID from hub registration to %s: %s", c.Address, err)
	}

	return parsedID, nil
}

// ListUsers is used to wrap the /list endpoint from the hub
func (c *Client) ListUsers() ([]uint64, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/list", c.Address))
	if err != nil {
		return nil, fmt.Errorf("failed to reach hub %s: %s", c.Address, err)
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %s: %s", c.Address, err)
	}

	response := &ListResponse{}
	if err := json.Unmarshal(b, response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response from %s: %s", c.Address, err)
	}

	return response.IDs, nil
}

func (c *Client) Identify() (uint64, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/identify?id=%d", c.Address, c.ID))
	if err != nil {
		return 0, fmt.Errorf("failed to reach hub %s: %s", c.Address, err)
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response from %s: %s", c.Address, err)
	}

	id, err := strconv.ParseUint(string(b), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("unable to parse response as uint64: %s", err)
	}

	return id, nil
}
