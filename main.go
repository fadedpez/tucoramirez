package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/tucobot"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file. Make sure it exists and is properly formatted.")
		return
	}

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		fmt.Println("No token provided. Please set the DISCORD_TOKEN environment variable in your .env file.")
		return
	}

	appID := os.Getenv("DISCORD_APP_ID")
	if appID == "" {
		fmt.Println("No application ID provided. Please set the DISCORD_APP_ID environment variable in your .env file.")
		return
	}

	guildID := os.Getenv("DISCORD_GUILD_ID") // Optional, leave empty to register commands globally

	// Create a new Discord session
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session:", err)
		return
	}

	// Add handlers
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		tucobot.InteractionCreate(s, i)
	})

	// Open websocket connection
	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening connection:", err)
		return
	}

	// Register commands
	tucobot.RegisterCommands(dg, appID, guildID)

	fmt.Println("Tuco is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Clean up
	dg.Close()
}
