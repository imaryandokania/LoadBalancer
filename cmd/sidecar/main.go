package main

import (
	"log"

	"try/pkg/server"
)

func main() {
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start sidecar proxy: %v", err)
	}
}
