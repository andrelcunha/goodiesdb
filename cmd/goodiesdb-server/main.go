package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"com.github.andrelcunha.goodiesdb/internal/core/server"
	"github.com/joho/godotenv"
)

var version string = "v0.0.1"

func main() {
	// Create a channel to listen for termination signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a context that is canceled when a termination signal is received
	ctx, cancel := context.WithCancel(context.Background())

	// Goroutine to handle termination signals
	go func() {
		<-signalChan
		cancel()
	}()

	// Load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file, using default values.")
	}
	// Set up configuration
	config := server.NewConfig()
	config.Version = version
	config.LoadFromEnv()

	// Initialize Server
	srv := server.NewServer(config)

	// Start the server
	go func() {
		if err := srv.Start(); err != nil {
			fmt.Println("Error starting server:", err)
			cancel()
		}
	}()

	// Block until the context is canceled
	<-ctx.Done()
	fmt.Println("\nReceived termination signal. ")
	fmt.Println("Shutting down Redis Clone Server...")
	srv.Shutdown()
}
