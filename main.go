package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"
	"time"
)

var Token string

func init() {
	flag.StringVar(&Token, "token", "", "Bot Token")
	flag.Parse()
}

func main() {
	if Token == "" {
		panic("Token is required")
	}

	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %s\n", s.State.User.Username)
	})

	dg.AddHandler(messageCreate)

	dg.Identify.Intents |= discordgo.IntentsGuildMembers
	dg.Identify.Intents |= discordgo.IntentsGuildMessageReactions
	dg.Identify.Intents |= discordgo.IntentsGuilds

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	defer func(dg *discordgo.Session) {
		err := dg.Close()
		if err != nil {
			fmt.Println("error closing connection,", err)
			return
		}
	}(dg)

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	for {
		select {
		case <-sc:
			return
		default:
		}
		time.Sleep(1 * time.Second)
	}
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
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
			_, _ = s.ChannelMessageSend(m.ChannelID, "Hurrah! Come back when you learn how to shoot cabrón! (Tuco: "+tucoString+" ; User: "+userString+")")
		} else if tucoRoll < userRoll {
			_, _ = s.ChannelMessageSend(m.ChannelID, "You pig! You haven't seen the last of Tuco! (Tuco: "+tucoString+" ; User: "+userString+")")
		} else {
			_, _ = s.ChannelMessageSend(m.ChannelID, "It seems we live to fight another day, friend. (Tuco: "+tucoString+" ; User: "+userString+")")
		}
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
