package tucobot

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/tucobot/games"
)

// Interface for Discord session operations
type sessionHandler interface {
	// Command registration
	ApplicationCommandCreate(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error)
	// Command listing
	ApplicationCommands(appID, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error)
	// Command deletion
	ApplicationCommandDelete(appID, guildID, cmdID string, options ...discordgo.RequestOption) error
	// Interaction responses
	InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
	// Message sending
	ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error)
}

// Commands
var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "dueltuco",
		Description: "Duel Tuco",
		// Ensure command is available in DMs and servers
		DMPermission: new(bool),
	},
	{
		Name:        "blackjack",
		Description: "Start a game of blackjack with Tuco",
		// Ensure command is available in DMs and servers
		DMPermission: new(bool),
	},
}

// RegisterCommands registers all slash commands with Discord
func RegisterCommands(s sessionHandler, appID string, guildID string) error {
	fmt.Printf("Starting command registration with appID: %s, guildID: %s\n", appID, guildID)

	// First, clean up any existing commands
	existingCommands, err := s.ApplicationCommands(appID, guildID)
	if err != nil {
		return fmt.Errorf("error fetching existing commands: %v", err)
	}

	fmt.Printf("Found %d existing commands\n", len(existingCommands))
	for _, cmd := range existingCommands {
		fmt.Printf("Deleting command: %s (%s)\n", cmd.Name, cmd.ID)
		if err := s.ApplicationCommandDelete(appID, guildID, cmd.ID); err != nil {
			fmt.Printf("Warning: error deleting command %s: %v\n", cmd.Name, err)
			// Continue even if deletion fails
		}
	}

	// Register new commands
	fmt.Printf("Registering %d commands\n", len(commands))
	registeredCommands := make(map[string]string)
	for _, cmd := range commands {
		fmt.Printf("Registering command: %s\n", cmd.Name)
		registeredCmd, err := s.ApplicationCommandCreate(appID, guildID, cmd)
		if err != nil {
			return fmt.Errorf("error registering command %s: %v", cmd.Name, err)
		}
		registeredCommands[registeredCmd.Name] = registeredCmd.ID
		fmt.Printf("Successfully registered command %s with ID: %s\n", registeredCmd.Name, registeredCmd.ID)
	}

	// Verify commands were registered
	verifyCommands, err := s.ApplicationCommands(appID, guildID)
	if err != nil {
		return fmt.Errorf("error verifying commands: %v", err)
	}

	fmt.Printf("Verifying %d commands\n", len(verifyCommands))
	for _, cmd := range verifyCommands {
		if id, exists := registeredCommands[cmd.Name]; exists {
			if id != cmd.ID {
				return fmt.Errorf("command %s has mismatched ID. Expected %s, got %s", cmd.Name, id, cmd.ID)
			}
			fmt.Printf("Verified command %s (ID: %s)\n", cmd.Name, cmd.ID)
		} else {
			return fmt.Errorf("unexpected command found: %s (ID: %s)", cmd.Name, cmd.ID)
		}
	}

	if len(verifyCommands) != len(commands) {
		return fmt.Errorf("command count mismatch. Expected %d, got %d", len(commands), len(verifyCommands))
	}

	return nil
}

// CleanupCommands removes all registered commands
func CleanupCommands(s sessionHandler, appID string, guildID string) {
	fmt.Printf("Starting command cleanup with appID: %s, guildID: %s\n", appID, guildID)

	existingCommands, err := s.ApplicationCommands(appID, guildID)
	if err != nil {
		fmt.Printf("Error fetching existing commands: %v\n", err)
		return
	}

	fmt.Printf("Found %d commands to clean up\n", len(existingCommands))
	for _, cmd := range existingCommands {
		fmt.Printf("Deleting command: %s (%s)\n", cmd.Name, cmd.ID)
		err := s.ApplicationCommandDelete(appID, guildID, cmd.ID)
		if err != nil {
			fmt.Printf("Error deleting command %s: %v\n", cmd.Name, err)
		}
	}

	fmt.Println("Cleanup complete!")
}

// InteractionCreate handles all incoming Discord interactions
func InteractionCreate(s sessionHandler, i *discordgo.InteractionCreate) {
	fmt.Printf("Received interaction type: %v\n", i.Type)

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		fmt.Printf("Received command: %s\n", i.ApplicationCommandData().Name)
		switch i.ApplicationCommandData().Name {
		case "dueltuco":
			handleDuelTuco(s, i)
		case "blackjack":
			handleBlackjack(s, i)
		default:
			fmt.Printf("Unknown command: %s\n", i.ApplicationCommandData().Name)
		}
	case discordgo.InteractionMessageComponent:
		fmt.Printf("Received button click: %s\n", i.MessageComponentData().CustomID)
		switch {
		case strings.HasPrefix(i.MessageComponentData().CustomID, "blackjack_"):
			games.HandleBlackjackButton(s, i)
		case i.MessageComponentData().CustomID == "duel_button":
			handleDuelButton(s, i)
		default:
			fmt.Printf("Unknown button: %s\n", i.MessageComponentData().CustomID)
		}
	}
}

func handleDuelTuco(s sessionHandler, i *discordgo.InteractionCreate) {
	buttons := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    "Duel!",
			Style:    discordgo.PrimaryButton,
			CustomID: "duel_button",
		},
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    "¿Quieres pelear, eh? Click the button below to duel!",
			Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: buttons}},
		},
	})
	if err != nil {
		fmt.Printf("Error responding to duel command: %v\n", err)
	}
}

func handleBlackjack(s sessionHandler, i *discordgo.InteractionCreate) {
	fmt.Printf("Starting blackjack game for user: %s\n", i.Member.User.Username)
	games.StartBlackjackGame(s, i)
}

func handleDuelButton(s sessionHandler, i *discordgo.InteractionCreate) {
	// Generate random number between 1 and 100
	rand.Seed(time.Now().UnixNano())
	userRoll := rand.Intn(100) + 1
	tucoRoll := rand.Intn(100) + 1

	var result string
	if userRoll > tucoRoll {
		result = fmt.Sprintf("You win! 🎉\nYou rolled %d\nTuco rolled %d", userRoll, tucoRoll)
	} else if userRoll < tucoRoll {
		result = fmt.Sprintf("Tuco wins! 😈\nYou rolled %d\nTuco rolled %d", userRoll, tucoRoll)
	} else {
		result = fmt.Sprintf("It's a tie! 🤝\nYou both rolled %d", userRoll)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: result,
		},
	})
	if err != nil {
		fmt.Printf("Error responding to duel button: %v\n", err)
	}
}

func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "tucosay" {
		quote, err := randFromTxt("quotes.txt")
		if err != nil {
			fmt.Println("Error getting random quote:", err)
			return
		}
		_, err = s.ChannelMessageSend(m.ChannelID, quote)
		if err != nil {
			fmt.Println("Error sending message:", err)
		}
	}

	if regexp.MustCompile(`[tT][hH][aA][nN][kK][sS] [tT][uU][cC][oO]`).MatchString(m.Content) {
		_, err := s.ChannelMessageSend(m.ChannelID, "De nada, amigo.")
		if err != nil {
			fmt.Println("Error sending message:", err)
		}
	}

	if m.Content == "tucoduel" {
		tucoRoll := diceRoll()
		userRoll := diceRoll()

		tucoString := strconv.Itoa(tucoRoll)
		userString := strconv.Itoa(userRoll)

		var content string
		if tucoRoll > userRoll {
			content = fmt.Sprintf("Hurrah! Come back when you learn how to shoot cabrón! (Tuco: %s ; %s: %s)", tucoString, m.Author.Mention(), userString)
		} else if tucoRoll < userRoll {
			content = fmt.Sprintf("You pig! You haven't seen the last of Tuco! (Tuco: %s ; %s: %s)", tucoString, m.Author.Mention(), userString)
		} else {
			content = fmt.Sprintf("It seems we live to fight another day, friend. (Tuco: %s ; %s: %s)", tucoString, m.Author.Mention(), userString)
		}

		_, err := s.ChannelMessageSend(m.ChannelID, content)
		if err != nil {
			fmt.Println("Error sending message:", err)
		}
	}

	if regexp.MustCompile(`tuco\?$`).MatchString(m.Content) {
		image, err := randFromTxt("images.txt")
		if err != nil {
			fmt.Println("Error getting random image:", err)
			return
		}
		_, err = s.ChannelMessageSend(m.ChannelID, image)
		if err != nil {
			fmt.Println("Error sending message:", err)
		}
	}
}

func randFromTxt(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var quotes []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		quotes = append(quotes, scanner.Text())
	}

	if len(quotes) == 0 {
		return "", fmt.Errorf("no quotes found in file %s", path)
	}

	quote := quotes[rand.Intn(len(quotes))]
	return quote, nil
}

func diceRoll() int {
	min := 1
	max := 100
	r := rand.Intn(max-min) + min
	return r
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
