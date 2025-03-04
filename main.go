package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/tucobot"
	"github.com/joho/godotenv"
)

func main() {
	// Print working directory
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting working directory: %v\n", err)
		return
	}
	fmt.Printf("Working directory: %s\n", wd)

	// Check if .env file exists
	envPath := filepath.Join(wd, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		fmt.Printf(".env file not found at: %s\n", envPath)
		return
	} else {
		fmt.Printf(".env file found at: %s\n", envPath)
	}

	// Load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Error loading .env file: %v\n", err)
		return
	}
	fmt.Println(".env file loaded successfully")

	// List all environment variables (excluding the actual values for security)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) > 0 {
			fmt.Printf("Found environment variable: %s\n", parts[0])
		}
	}

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		fmt.Println("No token provided. Please set the DISCORD_TOKEN environment variable in your .env file.")
		return
	}
	fmt.Printf("Token length: %d\n", len(token))
	fmt.Printf("Token first/last 4 chars: %s...%s\n", token[:4], token[len(token)-4:])
	if strings.HasPrefix(token, "Bot ") {
		fmt.Println("Warning: Token already has 'Bot ' prefix, removing it")
		token = strings.TrimPrefix(token, "Bot ")
	}

	appID := os.Getenv("DISCORD_APP_ID")
	if appID == "" {
		fmt.Println("No application ID provided. Please set the DISCORD_APP_ID environment variable in your .env file.")
		return
	}
	fmt.Printf("App ID: %s\n", appID)

	// GUILD_ID is optional and not needed for global command registration
	if guildID := os.Getenv("DISCORD_GUILD_ID"); guildID != "" {
		fmt.Printf("Warning: DISCORD_GUILD_ID is set but will be ignored. Commands will be registered globally.\n")
	}

	// Create a new Discord session
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Printf("Error creating Discord session: %v\n", err)
		return
	}

	// Add handlers
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		fmt.Printf("Received interaction from Discord\n")
		tucobot.InteractionCreate(s, i)
	})
	dg.AddHandler(tucobot.MessageCreate)

	// Set required intents for the bot
	dg.Identify.Intents = discordgo.IntentsGuildMessages | 
		discordgo.IntentsDirectMessages | 
		discordgo.IntentsGuildMessageReactions | 
		discordgo.IntentsGuildIntegrations |
		discordgo.IntentsGuilds

	// Open websocket connection
	err = dg.Open()
	if err != nil {
		fmt.Printf("Error opening connection: %v\n", err)
		return
	}

	// Register commands globally (empty guildID means global)
	fmt.Println("Registering commands globally...")
	if err := tucobot.RegisterCommands(dg, appID, ""); err != nil {
		fmt.Printf("Error registering commands: %v\n", err)
		dg.Close()
		return
	}
	fmt.Println("Commands registered successfully!")

	fmt.Println("Tuco is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Clean up
	fmt.Println("Cleaning up before exit...")
	tucobot.CleanupCommands(dg, appID, "")
	dg.Close()
}
