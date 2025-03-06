package bot

import (
	"github.com/bwmarrin/discordgo"
)

// Commands defines all slash commands for the bot
var Commands = []*discordgo.ApplicationCommand{
	{
		Name:        "blackjack",
		Description: "Start a game of blackjack",
	},
	{
		Name:        "dueltuco",
		Description: "Challenge Tuco to a duel",
	},
	// Add more commands here
}
