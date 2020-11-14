package hub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/StephenBirch/message-delivery-system/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHub_selfIdentify(t *testing.T) {
	tests := []struct {
		name              string
		expectedCode      int
		expectedError     gin.H
		inputID, outputID string
		clients           map[uint64]chan []byte
	}{
		{
			name:         "Golden Path",
			inputID:      "2387695293",
			outputID:     "2387695293",
			expectedCode: 200,
			clients: map[uint64]chan []byte{
				2387695293: make(chan []byte),
			},
		},
		{
			name:          "Client doesn't exist",
			inputID:       "2387695293",
			expectedCode:  400,
			expectedError: gin.H{"message": "ID not registered", "status": "Bad Request"},
		},
		{
			name:          "No ID given",
			expectedCode:  400,
			expectedError: gin.H{"message": "ID is required", "status": "Bad Request"},
			clients: map[uint64]chan []byte{
				2387695293: make(chan []byte),
			},
		},
		{
			name:          "ID given but not a uint64",
			expectedCode:  400,
			inputID:       "notuint64",
			expectedError: gin.H{"message": "strconv.ParseUint: parsing \"notuint64\": invalid syntax", "status": "Bad Request"},
			clients: map[uint64]chan []byte{
				2387695293: make(chan []byte),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := New()
			h.Clients = tt.clients

			req, err := http.NewRequest("GET", fmt.Sprintf("/identify?id=%s", tt.inputID), nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			h.Router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectedError != nil {
				var errorBody gin.H
				require.NoError(t, json.NewDecoder(w.Body).Decode(&errorBody))

				assert.Equal(t, tt.expectedError, errorBody)
				return
			}

			assert.Equal(t, tt.outputID, string(w.Body.Bytes()))
		})
	}
}

func TestHub_listUsers(t *testing.T) {
	tests := []struct {
		name           string
		expectedLength int
		expectedCode   int
		id             string
		clients        map[uint64]chan []byte
	}{
		{
			name:           "Single",
			expectedLength: 1,
			expectedCode:   200,
			clients: map[uint64]chan []byte{
				100: make(chan []byte),
			},
			id: "0",
		},
		{
			name:           "Double",
			expectedLength: 2,
			expectedCode:   200,
			clients: map[uint64]chan []byte{
				100: make(chan []byte),
				200: make(chan []byte),
			},
			id: "0",
		},
		{
			name:           "Double including self",
			expectedLength: 1,
			expectedCode:   200,
			clients: map[uint64]chan []byte{
				100: make(chan []byte),
				200: make(chan []byte),
			},
			id: "100",
		},
		{
			name:           "Just a coke",
			expectedLength: 0,
			expectedCode:   200,
			clients:        map[uint64]chan []byte{},
			id:             "0",
		},
		{
			name:           "No ID",
			expectedLength: 0,
			expectedCode:   400,
			clients:        map[uint64]chan []byte{},
		},
		{
			name:           "Invalid ID",
			expectedLength: 0,
			expectedCode:   400,
			clients:        map[uint64]chan []byte{},
			id:             "invalid",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New()
			h.Clients = tt.clients

			req, err := http.NewRequest("GET", fmt.Sprintf("/users?id=%s", tt.id), nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			h.Router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			var users types.ListResponse
			err = json.Unmarshal(w.Body.Bytes(), &users)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedLength, len(users.IDs))
		})
	}
}

func TestHub_register(t *testing.T) {
	tests := []struct {
		name          string
		expectedCode  int
		expectedError gin.H
	}{
		{
			name:         "Golden Path",
			expectedCode: 200,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := New()

			req, err := http.NewRequest("GET", "/register", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			h.Router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectedError != nil {
				var errorBody gin.H
				require.NoError(t, json.NewDecoder(w.Body).Decode(&errorBody))

				assert.Equal(t, tt.expectedError, errorBody)
				return
			}

			_, err = strconv.ParseUint(string(w.Body.Bytes()), 10, 64)
			require.NoError(t, err)

		})
	}
}

func TestHub_registerOwnID(t *testing.T) {
	tests := []struct {
		name          string
		expectedCode  int
		expectedError gin.H
		inputID       string
		outputID      uint64
		clients       map[uint64]chan []byte
	}{
		{
			name:         "Golden Path",
			expectedCode: 200,
			inputID:      "9001",
			outputID:     uint64(9001),
			clients:      map[uint64]chan []byte{},
		},
		{
			name:          "Not uint64 parsable",
			expectedCode:  400,
			inputID:       "notuint64",
			expectedError: gin.H{"message": "strconv.ParseUint: parsing \"notuint64\": invalid syntax", "status": "Bad Request"},
			clients:       map[uint64]chan []byte{},
		},
		{
			name:          "ID already exists",
			expectedCode:  400,
			inputID:       "500",
			expectedError: gin.H{"message": "ID already in use", "status": "Bad Request"},
			clients: map[uint64]chan []byte{
				500: make(chan []byte),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := New()

			h.Clients = tt.clients

			req, err := http.NewRequest("GET", fmt.Sprintf("/register?id=%s", tt.inputID), nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			h.Router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectedError != nil {
				var errorBody gin.H
				require.NoError(t, json.NewDecoder(w.Body).Decode(&errorBody))

				assert.Equal(t, tt.expectedError, errorBody)
				return
			}

			id, err := strconv.ParseUint(string(w.Body.Bytes()), 10, 64)
			require.NoError(t, err)

			assert.Equal(t, tt.outputID, id)
		})
	}
}

func TestHub_sendMessage(t *testing.T) {
	tests := []struct {
		name          string
		expectedCode  int
		expectedError gin.H
		inputID       string
		inputBody     io.Reader
		clients       map[uint64]chan []byte
	}{
		{
			name:         "Golden Path",
			expectedCode: 200,
			clients: map[uint64]chan []byte{
				500: make(chan []byte),
			},
			inputID:   "500",
			inputBody: bytes.NewBuffer([]byte("Hi")),
		},
		{
			name:         "No ids",
			expectedCode: 400,
			clients: map[uint64]chan []byte{
				500: make(chan []byte),
			},
			inputBody:     bytes.NewBuffer([]byte("Hi")),
			expectedError: gin.H{"message": "IDs are required (csv)", "status": "Bad Request"},
		},
		{
			name:         "No body",
			expectedCode: 400,
			clients: map[uint64]chan []byte{
				500: make(chan []byte),
			},
			inputID:       "500",
			expectedError: gin.H{"message": "Body expected for a sendmessage call", "status": "Bad Request"},
			inputBody:     nil,
		},
		{
			name:         "id not uint64",
			expectedCode: 400,
			clients: map[uint64]chan []byte{
				500: make(chan []byte),
			},
			inputID:       "notuint64",
			expectedError: gin.H{"message": "strconv.ParseUint: parsing \"notuint64\": invalid syntax", "status": "Bad Request"},
			inputBody:     bytes.NewBuffer([]byte("Hi")),
		},
		{
			name:          "no clients",
			expectedCode:  400,
			inputID:       "223154",
			expectedError: gin.H{"message": "ID not registered", "status": "Bad Request"},
			inputBody:     bytes.NewBuffer([]byte("Hi")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New()
			h.Clients = tt.clients

			req, err := http.NewRequest("POST", fmt.Sprintf("/send?ids=%s", tt.inputID), tt.inputBody)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			// go func needed since channels are used from within, needs to be threaded
			go func() { h.Router.ServeHTTP(w, req) }()

			// time for request to finish
			time.Sleep(time.Second)

			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectedError != nil {
				var errorBody gin.H
				require.NoError(t, json.NewDecoder(w.Body).Decode(&errorBody))
				assert.Equal(t, tt.expectedError, errorBody)
				return
			}
		})
	}
}

func TestHub_websocketInit(t *testing.T) {
	tests := []struct {
		name          string
		expectedCode  int
		expectedError gin.H
		inputID       string
		inputBody     types.SendingMessage
		clients       map[uint64]chan []byte
	}{
		{
			name:         "Golden Path",
			expectedCode: 200,
			clients: map[uint64]chan []byte{
				500: make(chan []byte),
			},
			inputID: "500",
			inputBody: types.SendingMessage{
				Recipients: "500",
				Data:       []byte("asdfbuyho"),
			},
		},
		{
			name:         "no id",
			expectedCode: 400,
			clients: map[uint64]chan []byte{
				500: make(chan []byte),
			},
			expectedError: gin.H{"message": "ID is required", "status": "Bad Request"},
		},
		{
			name:         "id not uint64",
			expectedCode: 400,
			clients: map[uint64]chan []byte{
				500: make(chan []byte),
			},
			expectedError: gin.H{"message": "strconv.ParseUint: parsing \"notuint64\": invalid syntax", "status": "Bad Request"},
			inputID:       "notuint64",
		},
		{
			name:         "id doesn't exist",
			expectedCode: 400,
			clients: map[uint64]chan []byte{
				500: make(chan []byte),
			},
			expectedError: gin.H{"message": "ID not registered", "status": "Bad Request"},
			inputID:       "200",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New()
			h.Clients = tt.clients

			// go func needed since channels are used from within, needs to be threaded
			go func() { h.Router.Run("localhost:8080") }()

			conn, resp, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://localhost:8080/ws?id=%s", tt.inputID), nil)
			require.Equal(t, tt.expectedError != nil, err != nil)

			if tt.expectedError != nil {
				var errorBody gin.H
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errorBody))
				assert.Equal(t, tt.expectedError, errorBody)
				return
			}

			// Error paths have returned here, try read & writes on the websocket conn
			b, err := json.Marshal(tt.inputBody)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", fmt.Sprintf("/send?ids=%s", tt.inputID), bytes.NewBuffer(b))
			require.NoError(t, err)

			w := httptest.NewRecorder()

			// go func needed since channels are used from within, needs to be threaded
			go func() { h.Router.ServeHTTP(w, req) }()

			time.Sleep(time.Second)

			assert.Equal(t, w.Code, 200)

			require.NoError(t, conn.WriteMessage(1, b))
		})
	}
}
