package tucobot

import (
	"bufio"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"math/rand"
	"os"
	"regexp"
	"strconv"
)

func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "tucosay" {
		_, _ = s.ChannelMessageSend(m.ChannelID, randFromTxt("quotes.txt"))
	}

	if regexp.MustCompile(`[tT][hH][aA][nN][kK][sS] [tT][uU][cC][oO]`).MatchString(m.Content) {
		_, _ = s.ChannelMessageSend(m.ChannelID, "De nada, amigo.")
	}

	if m.Content == "tucoduel" {
		tucoRoll := diceRoll()
		userRoll := diceRoll()

		tucoString := strconv.Itoa(tucoRoll)
		userString := strconv.Itoa(userRoll)

		if tucoRoll > userRoll {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Hurrah! Come back when you learn how to shoot cabrón! (Tuco: "+tucoString+" ; %s: "+userString+")")
		} else if tucoRoll < userRoll {
			_, _ = s.ChannelMessageSend(m.ChannelID, "You pig! You haven't seen the last of Tuco! (Tuco: "+tucoString+" ; %s: "+userString+")")
		} else {
			_, _ = s.ChannelMessageSend(m.ChannelID, "It seems we live to fight another day, friend. (Tuco: "+tucoString+" ; %s: "+userString+")")
		}
	}

	if regexp.MustCompile(`tuco\?$`).MatchString(m.Content) {
		_, _ = s.ChannelMessageSend(m.ChannelID, randFromTxt("images.txt"))
	}

}

func randFromTxt(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	var quotes []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		quotes = append(quotes, scanner.Text())
	}

	quote := quotes[rand.Intn(len(quotes))]
	return quote
}

func diceRoll() int {
	min := 1
	max := 100
	r := rand.Intn(max-min) + min
	return r
}

func RegisterCommands(s *discordgo.Session) {
	_, err := s.ApplicationCommandCreate(s.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "dueltuco",
		Description: "Duel Tuco",
	})

	if err != nil {
		fmt.Println("Cannot create command: ", err)
	}
}

func InteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		handleCommand(s, i)
	} else if i.Type == discordgo.InteractionMessageComponent {
		handleButtonClick(s, i)
	}

}

func handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ApplicationCommandData().Name == "dueltuco" {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Duel Tuco",
				Components: []discordgo.MessageComponent{
					&discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							&discordgo.Button{
								Label:    "Draw",
								Style:    discordgo.PrimaryButton,
								CustomID: "draw",
							},
						},
					},
				},
			},
		})
		if err != nil {
			fmt.Println("Cannot respond to command: ", err)
		}
	}
}

func handleButtonClick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.MessageComponentData().CustomID {
	case "draw":
		tucoRoll := diceRoll()
		userRoll := diceRoll()
		content := "che?"

		tucoString := strconv.Itoa(tucoRoll)
		userString := strconv.Itoa(userRoll)

		if tucoRoll > userRoll {
			content = "Hurrah! Come back when you learn how to shoot cabrón! (Tuco: " + tucoString + " ; %s: " + userString + ")"
		} else if tucoRoll < userRoll {
			content = "You pig! You haven't seen the last of Tuco! (Tuco: " + tucoString + " ; %s: " + userString + ")"
		} else {
			content = "It seems we live to fight another day, amigo. (Tuco: " + tucoString + " ; %s: " + userString + ")"
		}

		_, _ = s.ChannelMessageSend(i.ChannelID, content)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Duel Tuco",
		},
	})

	if err != nil {
		fmt.Println("Cannot respond to button click: ", err)
	}
}
