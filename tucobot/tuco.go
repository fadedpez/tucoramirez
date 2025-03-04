package tucobot

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/tucobot/games"
)

// Interface for Discord session operations
type sessionHandler interface {
	// Command registration
	ApplicationCommandCreate(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error)
	// Interaction responses
	InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
	// Message sending
	ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error)
	// Command deletion
	ApplicationCommandDelete(appID, guildID, cmdID string, options ...discordgo.RequestOption) error
}

// Commands
var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "dueltuco",
		Description: "Duel Tuco",
	},
	{
		Name:        "blackjack",
		Description: "Start a game of blackjack with Tuco",
	},
}

// RegisterCommands registers all slash commands with Discord
func RegisterCommands(s sessionHandler, appID string, guildID string) {
	fmt.Printf("Registering commands with appID: %s, guildID: %s\n", appID, guildID)
	for _, command := range commands {
		fmt.Printf("Registering command: %s\n", command.Name)
		cmd, err := s.ApplicationCommandCreate(appID, guildID, command)
		if err != nil {
			fmt.Printf("Cannot create command %s: %v\n", command.Name, err)
		} else {
			fmt.Printf("Successfully registered command %s with ID: %s\n", cmd.Name, cmd.ID)
		}
	}
}

// CleanupCommands removes all registered commands
func CleanupCommands(s *discordgo.Session, appID string, guildID string) {
	fmt.Println("Cleaning up commands...")
	registeredCommands, err := s.ApplicationCommands(appID, guildID)
	if err != nil {
		fmt.Printf("Could not fetch registered commands: %v\n", err)
		return
	}

	for _, cmd := range registeredCommands {
		err := s.ApplicationCommandDelete(appID, guildID, cmd.ID)
		if err != nil {
			fmt.Printf("Cannot delete command %s: %v\n", cmd.Name, err)
		} else {
			fmt.Printf("Successfully deleted command %s\n", cmd.Name)
		}
	}
	fmt.Println("Cleanup complete!")
}

// InteractionCreate handles all incoming Discord interactions
func InteractionCreate(s sessionHandler, i *discordgo.InteractionCreate) {
	fmt.Printf("Received interaction type: %v\n", i.Type)
	var err error
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		fmt.Printf("Received command: %s\n", i.ApplicationCommandData().Name)
		err = handleCommand(s, i)
	case discordgo.InteractionMessageComponent:
		fmt.Printf("Received button click: %s\n", i.MessageComponentData().CustomID)
		err = handleButtonClick(s, i)
	}
	if err != nil {
		fmt.Printf("Error handling interaction: %v\n", err)
	}
}

// Command handlers
func handleCommand(s sessionHandler, i *discordgo.InteractionCreate) error {
	switch i.ApplicationCommandData().Name {
	case "dueltuco":
		return handleDuelCommand(s, i)
	case "blackjack":
		return handleBlackjackCommand(s, i)
	default:
		return fmt.Errorf("unknown command %s", i.ApplicationCommandData().Name)
	}
}

func handleDuelCommand(s sessionHandler, i *discordgo.InteractionCreate) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "So another bastard wants to take out Tuco. Everyone wants to take out Tuco! You better hope you win because no one shoots at Tuco and lives to tell about it!",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Draw!",
							CustomID: "duel_button",
							Style:    discordgo.PrimaryButton,
						},
					},
				},
			},
		},
	})
}

func handleBlackjackCommand(s sessionHandler, i *discordgo.InteractionCreate) error {
	if err := games.StartBlackjackGame(s, i.ChannelID); err != nil {
		return err
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "¡Vamos a jugar blackjack! (Let's play blackjack!)",
		},
	})
}

// Button handlers
func handleButtonClick(s sessionHandler, i *discordgo.InteractionCreate) error {
	switch i.MessageComponentData().CustomID {
	case "duel_button":
		return handleDuelButton(s, i)
	case "join_button", "deal_button", "hit_button", "stand_button":
		return games.HandleButton(s, i)
	default:
		return fmt.Errorf("unknown button %s", i.MessageComponentData().CustomID)
	}
}

func handleDuelButton(s sessionHandler, i *discordgo.InteractionCreate) error {
	min := 1
	max := 100
	tucoRoll := min + rand.Intn(max-min)
	userRoll := min + rand.Intn(max-min)
	userMention := fmt.Sprintf("<@%s>", i.Member.User.ID)
	tucoString := strconv.Itoa(tucoRoll)
	userString := strconv.Itoa(userRoll)

	var content string
	if tucoRoll > userRoll {
		content = fmt.Sprintf("Hurrah! Come back when you learn how to shoot cabrón! (Tuco: %s ; %s: %s)", tucoString, userMention, userString)
	} else if tucoRoll < userRoll {
		content = fmt.Sprintf("You pig! You haven't seen the last of Tuco! (Tuco: %s ; %s: %s)", tucoString, userMention, userString)
	} else {
		content = fmt.Sprintf("It seems we live to fight another day, amigo. (Tuco: %s ; %s: %s)", tucoString, userMention, userString)
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
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
