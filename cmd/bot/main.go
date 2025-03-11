package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/fadedpez/tucoramirez/pkg/discord"
)

func main() {
	// Get bot token from environment
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("No Discord token provided. Set DISCORD_TOKEN environment variable")
	}

	// Create and start the bot
	bot, err := discord.New(token)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Start the bot
	if err := bot.Start(); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}

	log.Println("Bot is running. Press Ctrl+C to exit")

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	// Graceful shutdown
	log.Println("Shutting down...")
	if err := bot.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}
