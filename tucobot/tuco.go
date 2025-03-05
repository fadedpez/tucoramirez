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
	// Follow-up messages
	FollowupMessageCreate(interaction *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error)
	// Edit interaction response
	InteractionResponseEdit(i *discordgo.Interaction, edit *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error)
}

// Commands that are registered with Discord
var Commands = []*discordgo.ApplicationCommand{
	{
		Name:        "dueltuco",
		Description: "Challenge Tuco to a duel",
	},
	{
		Name:        "blackjack",
		Description: "Start a game of blackjack",
	},
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}

// Helper function to create int64 pointer
func ptr(i int64) *int64 {
	return &i
}

// RegisterCommands registers all slash commands with Discord
func RegisterCommands(s sessionHandler, appID string, guildID string) error {
	fmt.Printf("Starting command registration with appID: %s, guildID: %s\n", appID, guildID)
	fmt.Printf("Attempting to register %d commands\n", len(Commands))

	// First, clean up any existing commands
	existingCommands, err := s.ApplicationCommands(appID, guildID)
	if err != nil {
		return fmt.Errorf("error fetching existing commands: %v", err)
	}

	fmt.Printf("Found %d existing commands\n", len(existingCommands))
	for _, existingCmd := range existingCommands {
		fmt.Printf("Found existing command: %s (%s)\n", existingCmd.Name, existingCmd.ID)
		if err := s.ApplicationCommandDelete(appID, guildID, existingCmd.ID); err != nil {
			fmt.Printf("Error deleting command %s: %v\n", existingCmd.Name, err)
			return fmt.Errorf("error deleting command %s: %w", existingCmd.Name, err)
		}
		fmt.Printf("Successfully deleted command: %s\n", existingCmd.Name)
	}

	// Start registering new commands
	fmt.Println("Starting registration of commands")
	registeredCommands := make(map[string]string)
	var registrationErrors []string

	for _, cmd := range Commands {
		fmt.Printf("Attempting to register command: %s\n", cmd.Name)
		registeredCmd, err := s.ApplicationCommandCreate(appID, guildID, cmd)
		if err != nil {
			errMsg := fmt.Sprintf("error registering command %s: %v", cmd.Name, err)
			fmt.Printf("%s\n", errMsg)
			registrationErrors = append(registrationErrors, errMsg)
			continue
		}
		registeredCommands[cmd.Name] = registeredCmd.ID
		fmt.Printf("Successfully registered command %s with ID: %s\n", cmd.Name, registeredCmd.ID)
	}

	// Verify all commands were registered
	fmt.Println("Verifying registered commands...")
	verifiedCommands, err := s.ApplicationCommands(appID, guildID)
	if err != nil {
		return fmt.Errorf("error verifying registered commands: %w", err)
	}

	for _, cmd := range verifiedCommands {
		fmt.Printf("Found registered command: %s (%s)\n", cmd.Name, cmd.ID)
		if id, ok := registeredCommands[cmd.Name]; ok && id == cmd.ID {
			fmt.Printf("Verified command %s (ID: %s)\n", cmd.Name, cmd.ID)
		} else {
			errMsg := fmt.Sprintf("command %s failed verification", cmd.Name)
			registrationErrors = append(registrationErrors, errMsg)
		}
	}

	// Report final status
	successCount := len(registeredCommands)
	if successCount == len(Commands) {
		fmt.Printf("Successfully registered all %d commands\n", successCount)
		return nil
	}

	if len(registrationErrors) > 0 {
		fmt.Printf("Registered %d/%d commands with %d errors\n", successCount, len(Commands), len(registrationErrors))
		return fmt.Errorf("some commands failed to register: %v", registrationErrors)
	}

	return fmt.Errorf("unknown error during command registration")
}

// CleanupCommands removes all registered commands
func CleanupCommands(s sessionHandler, appID string, guildID string) error {
	fmt.Printf("Starting command cleanup with appID: %s, guildID: %s\n", appID, guildID)

	existingCommands, err := s.ApplicationCommands(appID, guildID)
	if err != nil {
		fmt.Printf("Error fetching existing commands: %v\n", err)
		return fmt.Errorf("error fetching existing commands: %w", err)
	}

	fmt.Printf("Found %d commands to clean up\n", len(existingCommands))
	for _, cmd := range existingCommands {
		fmt.Printf("Deleting command: %s (%s)\n", cmd.Name, cmd.ID)
		err := s.ApplicationCommandDelete(appID, guildID, cmd.ID)
		if err != nil {
			fmt.Printf("Warning: error deleting command %s: %v\n", cmd.Name, err)
			// Continue with other commands even if one fails
			continue
		}
	}

	fmt.Println("Cleanup complete!")
	return nil
}

func InteractionCreate(s sessionHandler, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		switch i.ApplicationCommandData().Name {
		case "blackjack":
			fmt.Printf("Starting blackjack game for user: %s\n", i.Member.User.Username)
			games.StartBlackjackGame(s, i)
		case "dueltuco":
			handleDuelTuco(s, i)
		}
	case discordgo.InteractionMessageComponent:
		if strings.HasPrefix(i.MessageComponentData().CustomID, "blackjack_") {
			games.HandleBlackjackButton(s, i)
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
