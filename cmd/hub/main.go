package main

import (
	"flag"
	"fmt"

	"github.com/StephenBirch/message-delivery-system/hub"
)

func main() {
	port := flag.Int("port", 8080, "The port where the hub will be exposed")
	flag.Parse()

	h := hub.New()
	h.Router.Run(fmt.Sprintf(":%d", *port))
}
