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

	// Set required intents for the bot
	dg.Identify.Intents = discordgo.IntentsAllWithoutPrivileged |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsMessageContent

	// Add handlers BEFORE opening the websocket
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Printf("\nBot is ready! Logged in as: %s#%s\n", s.State.User.Username, s.State.User.Discriminator)
		fmt.Printf("Bot ID: %s\n", s.State.User.ID)
	})

	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		fmt.Printf("\nReceived interaction from Discord\n")
		fmt.Printf("Type: %v\n", i.Type)
		if i.GuildID == "" {
			fmt.Printf("Warning: Received interaction without guild ID\n")
		} else {
			fmt.Printf("Guild ID: %s\n", i.GuildID)
		}
		if i.Member != nil && i.Member.User != nil {
			fmt.Printf("User: %s\n", i.Member.User.Username)
		}
		if i.Type == discordgo.InteractionApplicationCommand {
			fmt.Printf("Command name: %s\n", i.ApplicationCommandData().Name)
		}
		tucobot.InteractionCreate(s, i)
	})

	dg.AddHandler(func(s *discordgo.Session, g *discordgo.GuildCreate) {
		fmt.Printf("\nJoined guild: %s (ID: %s)\n", g.Name, g.ID)
		
		// Check bot permissions in guild
		for _, member := range g.Members {
			if member.User.ID == s.State.User.ID {
				fmt.Printf("Bot permissions in guild: %v\n", member.Permissions)
				break
			}
		}

		// Register commands for this guild
		if err := tucobot.RegisterCommands(s, appID, g.ID); err != nil {
			fmt.Printf("Error registering commands for guild %s: %v\n", g.Name, err)
		}
	})

	// Open websocket connection
	fmt.Println("\nOpening Discord connection...")
	err = dg.Open()
	if err != nil {
		fmt.Printf("Error opening connection: %v\n", err)
		return
	}
	fmt.Printf("Connection opened successfully for bot: %s\n", dg.State.User.Username)

	// Get list of guilds the bot is in
	guilds, err := dg.UserGuilds(100, "", "", false)
	if err != nil {
		fmt.Printf("Error getting guild list: %v\n", err)
		return
	}

	// Register commands for each guild
	fmt.Printf("\nFound %d guilds to register commands for\n", len(guilds))
	
	// First, try to clean up any global commands
	fmt.Println("Cleaning up any existing global commands...")
	if err := tucobot.CleanupCommands(dg, appID, ""); err != nil {
		fmt.Printf("Warning: error cleaning up global commands: %v\n", err)
	}

	// Then register for each guild
	var registrationErrors []string
	for _, g := range guilds {
		fmt.Printf("\nProcessing guild: %s (ID: %s)\n", g.Name, g.ID)
		
		// First cleanup any existing commands in this guild
		fmt.Printf("Cleaning up commands for guild %s...\n", g.Name)
		if err := tucobot.CleanupCommands(dg, appID, g.ID); err != nil {
			fmt.Printf("Warning: error cleaning up commands for guild %s: %v\n", g.Name, err)
			// Continue anyway as the registration might still work
		}

		// Then register new commands
		fmt.Printf("Registering commands for guild %s...\n", g.Name)
		if err := tucobot.RegisterCommands(dg, appID, g.ID); err != nil {
			errMsg := fmt.Sprintf("Error registering commands for guild %s: %v", g.Name, err)
			fmt.Printf("%s\n", errMsg)
			registrationErrors = append(registrationErrors, errMsg)
			// Continue with other guilds even if one fails
			continue
		}
		fmt.Printf("Successfully registered commands for guild: %s\n", g.Name)
	}

	if len(registrationErrors) > 0 {
		fmt.Printf("\nWarning: Encountered %d errors during command registration:\n", len(registrationErrors))
		for _, err := range registrationErrors {
			fmt.Printf("- %s\n", err)
		}
		fmt.Println("\nThe bot will continue running, but some commands may not be available in all guilds.")
	} else {
		fmt.Println("\nSuccessfully registered commands in all guilds!")
	}

	fmt.Println("\nBot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanup before exit
	fmt.Println("Cleaning up commands before shutdown...")
	tucobot.CleanupCommands(dg, appID, "") // We can ignore the error during shutdown
	
	// Close Discord session
	dg.Close()
}
