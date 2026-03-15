package main

import (
	"log"

	"github.com/alvarolobato/agent-cli/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		log.Fatal(err)
	}
}
