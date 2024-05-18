package main

import (
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/tucobot"
	"log"
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

	// dg.AddHandler(messageCreate) // This is the old way of doing things

	dg.AddHandler(tucobot.MessageCreate)

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
