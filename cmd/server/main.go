package main

import (
	"log"

	"github.com/junkd0g/sharingan/internal/tools"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := server.NewMCPServer(
		"sharingan",
		"1.0.0",
	)

	tools.Register(s)

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
