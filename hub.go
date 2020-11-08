package hub

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
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
	router.GET("/identify", h.identityMessage)
	router.GET("/stream", h.stream)
	router.POST("/send", h.sendMessage)

	return router
}

func (h *Hub) identityMessage(c *gin.Context) {
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

func (h *Hub) listUsers(c *gin.Context) {
	var users []uint64
	for userid := range h.Clients {
		users = append(users, userid)
	}
	c.JSON(http.StatusOK, users)
}

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
			c.SSEvent(fmt.Sprintf("%d", parsedID), msg)
			return true
		}
	})
}

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

		ch <- b
	}
}

func (h *Hub) idInUse(id uint64) bool {
	if _, exists := h.Clients[id]; !exists {
		return true
	}
	return false
}
