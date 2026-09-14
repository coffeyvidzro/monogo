package main

import (
	"log"

	"github.com/leamout/leamout/cmd/hosted/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		log.Fatal(err)
	}
}
