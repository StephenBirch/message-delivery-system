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
	h.Router.Run(fmt.Sprintf(":%d", port))
}
