package hub

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/StephenBirch/message-delivery-system/client"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Hub struct {
	sync.Mutex
	Router  *gin.Engine
	Clients map[uint64]chan []byte
}

func New() *Hub {
	h := &Hub{
		Clients: make(map[uint64]chan []byte),
	}
	h.Router = h.setup()

	return h
}

func (h *Hub) setup() *gin.Engine {
	router := gin.Default()

	router.GET("/list", h.listUsers)
	router.GET("/register", h.register)
	router.GET("/stream", h.stream)
	router.GET("/identify", h.selfIdentify)
	router.GET("/ws", h.websocketInit)
	router.POST("/send", h.sendMessage)
	router.GET("/tester", func(c *gin.Context) {
		wshandler(c.Writer, c.Request)
	})

	return router
}

// register takes an optional query "id", returns back the client id if its available, otherwise generates a random one.
func (h *Hub) register(c *gin.Context) {
	// If they don't provide an id, generate a random one
	if c.Query("id") == "" {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		newID := r.Uint64()
		for attempts := 0; !h.idInUse(newID); attempts++ {
			if attempts > 10 {
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

	h.Clients[newID] = make(chan []byte)
	c.JSON(http.StatusOK, newID)
}

// listUsers returns back an array of all userID's in use
func (h *Hub) listUsers(c *gin.Context) {
	var users client.ListResponse
	for userid := range h.Clients {
		users.IDs = append(users.IDs, userid)
	}

	c.JSON(http.StatusOK, users)
}

// stream is where the client will connect in to to get relayed all the messages from other users. Uses the http streaming functionality
func (h *Hub) stream(c *gin.Context) {
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

	closed := c.Writer.CloseNotify()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-closed:
			return false
		case msg := <-h.Clients[parsedID]:
			c.JSON(http.StatusOK, msg)
			// c.SSEvent(fmt.Sprintf("%d", parsedID), msg)
			return true
		}
	})
}

// sendMessages takes csv of clientIDs, and a Body containing byte array. It then puts the byte array in the channel of each client.
func (h *Hub) sendMessage(c *gin.Context) {
	if c.Query("ids") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Bad Request", "message": "IDs are required (csv)"})
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

func (h *Hub) websocketInit(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		//TODO
	}

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

	for {
		t, msg, err := conn.ReadMessage()
		if err != nil {
			//TODO
		}

		var incomingMessage client.SendingMessage
		err = json.Unmarshal(msg, incomingMessage)
		if err != nil {
			//TODO'
		}

		ids := strings.Split(incomingMessage.Recipients, ",")

		for _, id := range ids {
			//TODO From here tomorrow
		}
	}

	// select {
	// case msg := <-h.Clients[parsedID]:
	// }
	// c.JSON(http.StatusOK, msg)
	// // c.SSEvent(fmt.Sprintf("%d", parsedID), msg)
	// return true

}

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func wshandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsupgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Failed to set websocket upgrade: %+v", err)
		return
	}

	for {
		t, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		fmt.Println("Received: ", msg)
		conn.WriteMessage(t, msg)
	}
}
