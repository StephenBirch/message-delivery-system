package client

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/StephenBirch/message-delivery-system/hub"
	"github.com/StephenBirch/message-delivery-system/types"
	"github.com/stretchr/testify/require"
)

func TestHub_NewClient(t *testing.T) {
	tests := []struct {
		name          string
		hubRunning    bool
		expectedError bool
	}{
		{
			name:       "Golden Path",
			hubRunning: true,
		},
		{
			name:          "Client not running",
			expectedError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := hub.New()

			// wrap in a http.Server so we can force shutdown later
			serv := &http.Server{
				Addr:    ":8080",
				Handler: h.Router,
			}

			if tt.hubRunning {
				go func() {
					serv.ListenAndServe()
				}()
			}

			c, err := New("localhost:8080")
			require.Equal(t, tt.expectedError, err != nil)

			if !tt.expectedError {
				require.NotNil(t, c)
			}

			if tt.expectedError {
				require.Error(t, err)
			}

			serv.Shutdown(context.Background())
		})
	}
}

func TestHub_Identify(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Golden Path",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := hub.New()

			// wrap in a http.Server so we can force shutdown later
			serv := &http.Server{
				Addr:    ":8080",
				Handler: h.Router,
			}

			go func() {
				serv.ListenAndServe()
			}()

			c, err := New("localhost:8080")
			require.NoError(t, err)

			id, err := c.Identify()
			require.NoError(t, err)
			require.Equal(t, id, c.ID)

			serv.Shutdown(context.Background())
		})
	}
}

func TestHub_ListUsers(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Golden Path",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := hub.New()

			// wrap in a http.Server so we can force shutdown later
			serv := &http.Server{
				Addr:    ":8080",
				Handler: h.Router,
			}

			go func() {
				serv.ListenAndServe()
			}()

			c, err := New("localhost:8080")
			require.NoError(t, err)

			users, err := c.ListUsers()
			require.NoError(t, err)
			require.Equal(t, len(users.IDs), 1)

			serv.Shutdown(context.Background())
		})
	}
}

func TestVerifyRecipients(t *testing.T) {

	tests := []struct {
		name       string
		recipients string
		wantErr    bool
	}{
		{
			name:       "Single",
			recipients: "12341234",
		},
		{
			name:       "Single with trailing comma",
			recipients: "12341234,",
			wantErr:    true,
		},
		{
			name:       "Double",
			recipients: "12341234,21367894",
		},
		{
			name:       "Empty",
			recipients: "",
			wantErr:    true,
		},
		{
			name:       ">255 recipients somehow",
			recipients: "1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0",
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := VerifyRecipients(tt.recipients); (err != nil) != tt.wantErr {
				t.Errorf("VerifyRecipients() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerifyFile(t *testing.T) {

	tests := []struct {
		name     string
		filepath string
		wantErr  bool
	}{
		{
			name:     "Golden Path",
			filepath: "../exampleData/small.txt",
		},
		{
			name:     "doesn't exist",
			filepath: "../exampleData/medium.txt",
			wantErr:  true,
		},
		{
			name:     "Too big",
			filepath: "../exampleData/big.txt",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := VerifyFile(tt.filepath); (err != nil) != tt.wantErr {
				t.Errorf("VerifyFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHub_InitWebsocket(t *testing.T) {
	tests := []struct {
		name          string
		expectedError bool
		changeID      bool
	}{
		{
			name: "Golden Path",
		},
		{
			name:          "Client doesn't exist",
			changeID:      true,
			expectedError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := hub.New()

			// wrap in a http.Server so we can force shutdown later
			serv := &http.Server{
				Addr:    ":8080",
				Handler: h.Router,
			}

			go func() {
				serv.ListenAndServe()
			}()

			c, err := New("localhost:8080")
			require.NoError(t, err)
			require.NotNil(t, c)

			fmt.Println(h.Clients)
			fmt.Println()

			if tt.changeID {
				c.ID = 0
			}

			conn, err := c.InitWebsocket()
			require.Equal(t, tt.expectedError, err != nil)

			if !tt.expectedError {
				conn.Close()
			}

			serv.Shutdown(context.Background())
		})
	}
}

func TestHub_WriteMessages(t *testing.T) {
	tests := []struct {
		name          string
		send          []byte
		resetConn     bool
		expectedError bool
	}{
		{
			name: "Golden Path",
			send: []byte("blarg"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Println(tt.name)
			h := hub.New()

			// wrap in a http.Server so we can force shutdown later
			serv := &http.Server{
				Addr:    ":8080",
				Handler: h.Router,
			}

			go func() {
				serv.ListenAndServe()
			}()

			c, err := New("localhost:8080")
			require.NoError(t, err)
			require.NotNil(t, c)

			fmt.Println(h.Clients)
			fmt.Println()

			conn, err := c.InitWebsocket()
			require.NoError(t, err)
			defer conn.Close()

			go func() {
				if err := c.WriteMessages(conn); err != nil {
					t.Fatalf("Unexpected Error")
				}
			}()

			c.Sending <- types.SendingMessage{Recipients: fmt.Sprint(c.ID), Data: []byte(tt.send)}

			serv.Shutdown(context.Background())
		})
	}
}
