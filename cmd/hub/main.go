package main

import (
	"fmt"

	hub "github.com/StephenBirch/message-delivery-system"
)

var (
	port = 8080
)

func main() {
	h := hub.New()

	h.Router.Run(fmt.Sprintf(":%d", port))
}
