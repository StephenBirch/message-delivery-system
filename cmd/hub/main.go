package main

import (
	"fmt"

	"github.com/StephenBirch/message-delivery-system/hub"
)

var (
	port = 8080
)

func main() {
	h := hub.New()

	// go func() {
	// 	for {
	// 		time.Sleep(time.Second * 5)

	// 		c, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/test", nil)
	// 		if err != nil {
	// 			log.Fatal("dial:", err)
	// 		}
	// 		defer c.Close()

	// 		err = c.WriteMessage(websocket.TextMessage, []byte("test"))
	// 		if err != nil {
	// 			log.Println("write:", err)
	// 			return
	// 		}

	// 		_, message, err := c.ReadMessage()
	// 		if err != nil {
	// 			log.Println("read:", err)
	// 			return
	// 		}
	// 		log.Printf("recv: %s", message)
	// 	}
	// }()

	h.Router.Run(fmt.Sprintf(":%d", port))
}
