package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/fadedpez/tucoramirez/pkg/discord"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("¡Ay caramba! *frantically searches pockets* Where is my .env file, eh? Make sure it exists at the project root! 🎲")
	}

	// Get Discord token from environment
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN not set in environment")
	}

	// Create new bot instance
	bot, err := discord.NewBot(token)
	if err != nil {
		log.Fatalf("Error creating bot: %v", err)
	}

	// Start the bot
	if err := bot.Start(); err != nil {
		log.Fatalf("Error starting bot: %v", err)
	}

	log.Println("Bot is running. Press Ctrl+C to exit")

	// Wait for interrupt signal to gracefully shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	// Cleanup and exit
	log.Println("Shutting down...")
	if err := bot.Stop(); err != nil {
		log.Printf("Error stopping bot: %v", err)
	}
}
