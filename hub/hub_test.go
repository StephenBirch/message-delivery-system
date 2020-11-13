package hub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/StephenBirch/message-delivery-system/client"
	"github.com/gin-gonic/gin"
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
		clients        map[uint64]chan []byte
	}{
		{
			name:           "Single",
			expectedLength: 1,
			expectedCode:   200,
			clients: map[uint64]chan []byte{
				100: make(chan []byte),
			},
		},
		{
			name:           "Double",
			expectedLength: 2,
			expectedCode:   200,
			clients: map[uint64]chan []byte{
				100: make(chan []byte),
				200: make(chan []byte),
			},
		},
		{
			name:           "Just a coke",
			expectedLength: 0,
			expectedCode:   200,
			clients:        map[uint64]chan []byte{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New()
			h.Clients = tt.clients

			req, err := http.NewRequest("GET", "/users", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			h.Router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			var users client.ListResponse
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
