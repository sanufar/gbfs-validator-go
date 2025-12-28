// Command validator runs the GBFS validator as a CLI or HTTP API.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gbfs-validator-go/pkg/api"
	"github.com/gbfs-validator-go/pkg/env"
	"github.com/gbfs-validator-go/pkg/fetcher"
	"github.com/gbfs-validator-go/pkg/validator"
)

// main parses flags and chooses CLI or server mode.
func main() {
	if err := env.LoadFile(".env"); err != nil {
		log.Printf("Failed to load .env: %v", err)
	}

	var (
		port         = flag.Int("port", 8080, "Port to listen on")
		url          = flag.String("url", "", "GBFS feed URL to validate (CLI mode)")
		version      = flag.String("version", "", "Force specific GBFS version")
		docked       = flag.Bool("docked", false, "Require station-based (docked) files")
		freefloating = flag.Bool("freefloating", false, "Require free-floating vehicle files")
		lenient      = flag.Bool("lenient", false, "Enable lenient mode (coerce 0/1 to bool, string to number, etc.)")
	)
	flag.Parse()

	if *url != "" {
		runCLI(*url, *version, *docked, *freefloating, *lenient)
		return
	}

	runServer(*port)
}

// runCLI validates a feed URL and prints results to stdout.
func runCLI(feedURL, ver string, docked, freefloating, lenient bool) {
	fmt.Printf("Validating GBFS feed: %s\n", feedURL)
	if lenient {
		fmt.Println("Mode: LENIENT (data coercion enabled)")
	}
	fmt.Println("================================")

	f := fetcher.New()
	v := validator.New(f, validator.Options{
		Version:      ver,
		Docked:       docked,
		Freefloating: freefloating,
		LenientMode:  lenient,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := v.Validate(ctx, feedURL)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	fmt.Printf("\nVersion: detected=%s, validated=%s\n",
		result.Summary.Version.Detected,
		result.Summary.Version.Validated)

	if result.Summary.HasErrors {
		fmt.Printf("Status: INVALID (%d errors)\n", result.Summary.ErrorsCount)
	} else {
		fmt.Println("Status: VALID")
	}

	if lenient && result.Summary.CoercionSummary != nil && result.Summary.CoercionSummary.TotalCoercions > 0 {
		fmt.Printf("\nCoercions applied: %d\n", result.Summary.CoercionSummary.TotalCoercions)
	}

	fmt.Println("\nFiles:")
	for _, file := range result.Files {
		status := "✓"
		if file.HasErrors {
			status = "✗"
		} else if !file.Exists {
			if file.Required {
				status = "✗ MISSING (required)"
			} else {
				status = "- (optional, not present)"
			}
		}

		coercionInfo := ""
		if file.CoercionCount > 0 {
			coercionInfo = fmt.Sprintf(" [%d coercions]", file.CoercionCount)
		}

		fmt.Printf("  %s %s%s\n", status, file.File, coercionInfo)
		
		if file.HasErrors {
			// Limit error output to first 5 unique error types
			seen := make(map[string]int)
			for _, err := range file.Errors {
				key := err.Message
				seen[key]++
				if seen[key] == 1 && len(seen) <= 5 {
					fmt.Printf("      %s: %s\n", err.Severity, err.Message)
				}
			}
			if len(seen) > 5 {
				fmt.Printf("      ... and %d more unique error types\n", len(seen)-5)
			}
		}
	}

	if result.Summary.HasErrors {
		os.Exit(1)
	}
}

// runServer starts the HTTP API server with graceful shutdown.
func runServer(port int) {
	server := api.NewServer()

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      server,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Server is shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	log.Printf("GBFS Validator API server starting on port %d", port)
	log.Printf("API endpoints:")
	log.Printf("  POST /api/validator        - Validate a GBFS feed")
	log.Printf("  POST /api/feed             - Get feed data for visualization")
	log.Printf("  POST /api/validator-summary - Get grouped validation summary")
	log.Printf("  GET  /health               - Health check")

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on port %d: %v\n", port, err)
	}

	<-done
	log.Println("Server stopped")
}
