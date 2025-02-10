package main

import (
	"fmt"

	"github.com/traPtitech/Checkin-Server/router"
)

var (
	port = 3000
)

func main() {
	r := router.Handlers{}
	e := r.Setup()

	if err := e.Start(fmt.Sprintf(":%d", port)); err != nil {
		e.Logger.Info("shutting down the server")
	}
}
