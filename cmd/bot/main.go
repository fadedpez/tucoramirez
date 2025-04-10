package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fadedpez/tucoramirez/pkg/discord"
	"github.com/fadedpez/tucoramirez/pkg/repositories/game"
	walletRepo "github.com/fadedpez/tucoramirez/pkg/repositories/wallet"
	"github.com/fadedpez/tucoramirez/pkg/scheduler"
	"github.com/fadedpez/tucoramirez/pkg/services/statistics"
	walletService "github.com/fadedpez/tucoramirez/pkg/services/wallet"
	"github.com/joho/godotenv"
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

	// Create application context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Variable to hold the maintenance scheduler if we're using Elasticsearch
	var maintenanceScheduler *scheduler.ElasticsearchMaintenanceScheduler

	// You can use an environment variable to choose the repository type
	storageType := os.Getenv("STORAGE_TYPE") // Add this to your .env file
	// Ensure data directory exists for SQLite storage
	dataDir := "./data"

	if storageType == "sqlite" {
		// Ensure data directory exists
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
			// Check if Elasticsearch is configured
			esURL := os.Getenv("ELASTICSEARCH_URL")
			if esURL != "" {
				log.Printf("Elasticsearch URL configured: %s", esURL)
				esUsername := os.Getenv("ELASTICSEARCH_USERNAME")
				esPassword := os.Getenv("ELASTICSEARCH_PASSWORD")
				esIndexPrefix := os.Getenv("ELASTICSEARCH_INDEX_PREFIX")
				
				// Create Elasticsearch configuration
				esConfig := &game.ElasticsearchConfig{
					URL:         esURL,
					Username:    esUsername,
					Password:    esPassword,
					IndexPrefix: esIndexPrefix,
					ArchivePath: dataDir + "/archives",
					BackupPath:  dataDir + "/backups",
				}
				
				// Set retention period if configured
				if retentionDays := os.Getenv("ELASTICSEARCH_RETENTION_DAYS"); retentionDays != "" {
					var days int
					if _, err := fmt.Sscanf(retentionDays, "%d", &days); err == nil && days > 0 {
						esConfig.RetentionPeriod = time.Duration(days) * 24 * time.Hour
					}
				}
				
				// Set rotation period if configured
				if rotationDays := os.Getenv("ELASTICSEARCH_ROTATION_DAYS"); rotationDays != "" {
					var days int
					if _, err := fmt.Sscanf(rotationDays, "%d", &days); err == nil && days > 0 {
						esConfig.RotationPeriod = time.Duration(days) * 24 * time.Hour
					}
				} else {
					// Default to monthly rotation
					esConfig.RotationPeriod = 30 * 24 * time.Hour
				}
				
				// Configure backup settings
				if backupEnabled := os.Getenv("ELASTICSEARCH_BACKUP_ENABLED"); backupEnabled == "true" {
					esConfig.BackupEnabled = true
					
					// Set backup schedule if configured
					if backupSchedule := os.Getenv("ELASTICSEARCH_BACKUP_SCHEDULE"); backupSchedule != "" {
						esConfig.BackupSchedule = backupSchedule
					}
				}
				
				esRepo, err := game.NewElasticsearchRepository(sqliteRepo, esConfig)
				if err != nil {
					log.Printf("Failed to initialize Elasticsearch repository: %v", err)
					log.Println("Using SQLite repository only")
					gameRepo = sqliteRepo
				} else {
					log.Println("Successfully initialized Elasticsearch repository for statistics")
					gameRepo = esRepo
					
					// Initialize and start the maintenance scheduler
					log.Println("Initializing Elasticsearch maintenance scheduler")
					maintenanceScheduler = scheduler.NewElasticsearchMaintenanceScheduler(esRepo)
					maintenanceScheduler.Start(ctx)
				}
			} else {
				gameRepo = sqliteRepo
				log.Println("Successfully initialized SQLite repository for game data")
			}
		}
	} else {
		// Default to memory repository
		gameRepo = game.NewMemoryRepository()
		log.Println("Using in-memory repository for game data (data will be lost on restart)")
	}

	// Create new bot instance with repository
	// Initialize wallet repository
	var walletRepository walletRepo.Repository
	if storageType == "sqlite" {
		walletDbPath := dataDir + "/wallets.db"
		log.Printf("Initializing SQLite wallet repository at %s", walletDbPath)
		sqliteWalletRepo, err := walletRepo.NewSQLiteRepository(walletDbPath)
		if err != nil {
			log.Printf("Failed to initialize SQLite wallet repository: %v", err)
			log.Println("Falling back to in-memory wallet repository")
			walletRepository = walletRepo.NewMemoryRepository()
		} else {
			walletRepository = sqliteWalletRepo
			log.Println("Successfully initialized SQLite repository for wallet data")
		}
	} else {
		// Default to memory repository
		walletRepository = walletRepo.NewMemoryRepository()
		log.Println("Using in-memory repository for wallet data (data will be lost on restart)")
	}

	wService := walletService.NewService(walletRepository)
	
	// Initialize statistics service
	statsService := statistics.NewService(gameRepo)
	
	bot, err := discord.NewBot(token, gameRepo, wService, statsService)
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
	
	// Cancel the context to stop the scheduler
	cancel()
	
	// Stop the maintenance scheduler if it was started
	if maintenanceScheduler != nil {
		log.Println("Stopping Elasticsearch maintenance scheduler")
		maintenanceScheduler.Stop()
	}
	
	// Stop the bot
	if err := bot.Stop(); err != nil {
		log.Printf("Error stopping bot: %v", err)
	}
}
