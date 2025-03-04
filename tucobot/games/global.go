package games

import (
	"github.com/bwmarrin/discordgo"
)

func HandleJoinGame(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if _, ok := activeGames[i.ChannelID]; !ok {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No active game session",
			},
		})
		return
	}
}
