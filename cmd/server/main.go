// Command server starts the GBFS validator API server.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gbfs-validator-go/pkg/env"
	"github.com/gbfs-validator-go/pkg/api"
)

// main configures and runs the HTTP server.
func main() {
	if err := env.LoadFile(".env"); err != nil {
		log.Printf("Failed to load .env: %v", err)
	}

	port := flag.Int("port", 8080, "Server port")
	staticDir := flag.String("static", "", "Directory containing static files for viewer (optional)")
	flag.Parse()

	var server *api.Server
	
	if *staticDir != "" {
		// Check if directory exists
		if _, err := os.Stat(*staticDir); os.IsNotExist(err) {
			log.Fatalf("Static directory does not exist: %s", *staticDir)
		}
		server = api.NewServerWithStatic(*staticDir)
		log.Printf("Serving static files from: %s", *staticDir)
	} else {
		server = api.NewServer()
	}

	addr := fmt.Sprintf(":%d", *port)
	
	fmt.Println("┌─────────────────────────────────────────────┐")
	fmt.Println("│         GBFS VALIDATOR SERVER               │")
	fmt.Println("├─────────────────────────────────────────────┤")
	fmt.Printf("│  Server:    http://localhost%-15s │\n", addr)
	fmt.Println("│                                             │")
	fmt.Println("│  API Endpoints:                             │")
	fmt.Println("│    POST /api/validator                      │")
	fmt.Println("│    POST /api/validator-summary              │")
	fmt.Println("│    POST /api/feed                           │")
	fmt.Println("│    POST /api/gbfs                           │")
	fmt.Println("│    GET  /api/proxy?url=...                  │")
	fmt.Println("│    GET  /health                             │")
	if *staticDir != "" {
		fmt.Println("│                                             │")
		fmt.Println("│  Viewer:    http://localhost" + addr + "/            │")
	}
	fmt.Println("└─────────────────────────────────────────────┘")

	log.Fatal(http.ListenAndServe(addr, server))
}
