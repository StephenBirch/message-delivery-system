package hub

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/StephenBirch/message-delivery-system/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var maxAttempts = 5 // If somehow the uint64 is taken try this many times

// Hub struct represents a Hub, with both the Gin router and client map
type Hub struct {
	sync.Mutex
	Router  *gin.Engine
	Clients map[uint64]chan []byte
}

// New creates a Hub object, initing a map of all clients & setting the router up
func New() *Hub {
	h := &Hub{
		Clients: make(map[uint64]chan []byte),
	}
	h.Router = h.setup()

	return h
}

func (h *Hub) setup() *gin.Engine {
	router := gin.Default()

	router.GET("/register", h.register)
	router.GET("/ws", h.websocketInit)
	router.GET("/identify", h.selfIdentify)
	router.GET("/users", h.listUsers)

	router.POST("/send", h.sendMessage)

	return router
}

// register takes an optional query "id", returns back the client id if its available, otherwise generates a random one.
func (h *Hub) register(c *gin.Context) {
	// If they don't provide an id, generate a random one
	if c.Query("id") == "" {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		newID := r.Uint64()
		for attempts := 0; !h.idInUse(newID); attempts++ {
			if attempts > maxAttempts {
				c.JSON(http.StatusInternalServerError, gin.H{"status": "Internal Server Error", "message": "Failed to find ID not in use"})
				return
			}
			newID = r.Uint64()
		}

		h.Clients[newID] = make(chan []byte)
		c.JSON(http.StatusOK, newID)
		return
	}

	// If they provide an ID, check its an uint64
	newID, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Bad Request", "message": err.Error()})
		return
	}

	// Then check if its already in use
	if _, exists := h.Clients[newID]; exists {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Bad Request", "message": "ID already in use"})
		return
	}

	// Init a new channel for the ID
	h.Clients[newID] = make(chan []byte)

	c.JSON(http.StatusOK, newID)
}

// listUsers returns back an array of all userID's in use
func (h *Hub) listUsers(c *gin.Context) {
	if c.Query("id") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Bad Request", "message": "IDs is required"})
		return
	}

	parsedID, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Bad Request", "message": err.Error()})
		return
	}

	var users types.ListResponse
	for userid := range h.Clients {
		// We don't want to add our own ID
		if userid != parsedID {
			users.IDs = append(users.IDs, userid)
		}
	}

	c.JSON(http.StatusOK, users)
}

// sendMessages takes csv of clientIDs, and a Body containing byte array. It then puts the byte array in the channel of each types.
func (h *Hub) sendMessage(c *gin.Context) {
	if c.Query("ids") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Bad Request", "message": "IDs are required (csv)"})
		return
	}

	if c.Request.Body == nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Bad Request", "message": "Body expected for a sendmessage call"})
		return
	}

	b, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Bad Request", "message": "No JSON body found"})
		return
	}

	ids := strings.Split(c.Query("ids"), ",")

	if len(ids) > 255 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Bad Request", "message": "Maximum number of clients to send messages is 255"})
		return
	}

	for _, id := range ids {
		parsedID, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "Bad Request", "message": err.Error()})
			return
		}

		ch, exists := h.Clients[parsedID]
		if !exists || ch == nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "Bad Request", "message": "ID not registered"})
			return
		}

		b = append(b, byte('\n'))

		// Add the byte array onto the clients channel
		ch <- b
	}
}

// selfIdentify takes a query of an ID, it check that it exists and is valid. Returning back the ID if it is
// Note: this method is written as such since there's no authentication in this simple solution. If there was authentication via token etc,
// that would be used to maintain a map of userIDs to authentication method.
func (h *Hub) selfIdentify(c *gin.Context) {
	if c.Query("id") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Bad Request", "message": "ID is required"})
		return
	}

	parsedID, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Bad Request", "message": err.Error()})
		return
	}

	if ch, exists := h.Clients[parsedID]; !exists || ch == nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Bad Request", "message": "ID not registered"})
		return
	}

	c.JSON(http.StatusOK, parsedID)
}

// idInUse is used to check the client map to see if it exists
func (h *Hub) idInUse(id uint64) bool {
	if _, exists := h.Clients[id]; !exists {
		return true
	}
	return false
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// websocketInit starts & upgrades the connection to a websocket, then runs the reading and writing go funcs. Used for forwarding messages to the different clients.
func (h *Hub) websocketInit(c *gin.Context) {
	if c.Query("id") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Bad Request", "message": "ID is required"})
		return
	}

	connectedID, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Bad Request", "message": err.Error()})
		return
	}

	if ch, exists := h.Clients[connectedID]; !exists || ch == nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Bad Request", "message": "ID not registered"})
		return
	}

	// Upgrade connection to a websocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	// Handles incoming messages
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading message from %d: %v", connectedID, err)
				conn.Close()
				delete(h.Clients, connectedID)
				break
			}

			var incomingMessage types.SendingMessage
			err = json.Unmarshal(msg, &incomingMessage)
			if err != nil {
				log.Printf("Unable unmarshal message bound for %d: %v", connectedID, err)
				continue
			}

			ids := strings.Split(incomingMessage.Recipients, ",")

			for _, id := range ids {

				parsedID, err := strconv.ParseUint(id, 10, 64)
				if err != nil {
					log.Printf("Unable to parse recipient %v: %v", id, err)
					continue
				}

				h.Clients[parsedID] <- incomingMessage.Data
			}
		}
	}()

	// Handles outgoing messages
	go func() {
		for {
			select {
			case msg := <-h.Clients[connectedID]:
				err := conn.WriteMessage(1, msg)
				if err != nil {
					log.Printf("Error writing message to %d: %v", connectedID, err)
					conn.Close()
					delete(h.Clients, connectedID)
					break
				}
			}
		}
	}()

}
