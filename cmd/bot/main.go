package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/fadedpez/tucoramirez/pkg/discord"
	"github.com/fadedpez/tucoramirez/pkg/repositories/game"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("¡Ay caramba! *frantically searches pockets* Where is my .env file, eh? Make sure it exists at the project root! ")
	}

	// Get Discord token from environment
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN not set in environment")
	}

	// Initialize repository
	var gameRepo game.Repository
	
	// You can use an environment variable to choose the repository type
	storageType := os.Getenv("STORAGE_TYPE") // Add this to your .env file
	
	if storageType == "sqlite" {
		// Ensure data directory exists
		dataDir := "./data"
		log.Printf("Creating data directory at %s if it doesn't exist", dataDir)
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Fatalf("Failed to create data directory: %v", err)
		}
		
		dbPath := dataDir + "/tucoramirez.db"
		log.Printf("Initializing SQLite repository at %s", dbPath)
		sqliteRepo, err := game.NewSQLiteRepository(dbPath)
		if err != nil {
			log.Printf("Failed to initialize SQLite repository: %v", err)
			log.Println("Falling back to in-memory repository")
			gameRepo = game.NewMemoryRepository()
		} else {
			gameRepo = sqliteRepo
			log.Println("Successfully initialized SQLite repository for game data")
		}
	} else {
		// Default to memory repository
		gameRepo = game.NewMemoryRepository()
		log.Println("Using in-memory repository for game data (data will be lost on restart)")
	}

	// Create new bot instance with repository
	bot, err := discord.NewBot(token, gameRepo)
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
