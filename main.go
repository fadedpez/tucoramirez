package main

import (
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"math/rand"
	"os"
	"os/signal"
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

	var sayings = []string{
		"When you have to shoot, shoot, don't talk.",
		"There are two kinds of people in the world, my friend. Those who have a rope around their neck and those who have the job of doing the cutting. ",
		"You want to know who you are? Huh? Huh? You don't, I do, everyone does... you're the son of a thousand fathers, all bastards like you. ",
		"If you work for a living, why do you kill yourself working?",
		"There are two kinds of spurs, my friend. Those that come in by the door, those that come in by the window.",
		"Cervesa!",
	}

	if m.Content == "tucosay" {
		_, _ = s.ChannelMessageSend(m.ChannelID, sayings[rand.Intn(len(sayings))])
	}

}
