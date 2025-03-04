package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/tucobot"
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		fmt.Println("No token provided. Please set the DISCORD_TOKEN environment variable.")
		return
	}

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
	tucobot.RegisterCommands(dg)

	fmt.Println("Tuco is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Clean up
	dg.Close()
}
