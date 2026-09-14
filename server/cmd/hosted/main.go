package main

import (
	"log"

	"github.com/coffeyvidzro/monogo/cmd/hosted/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		log.Fatal(err)
	}
}
