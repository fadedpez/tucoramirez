package tucobot

import (
	"bufio"
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
