package games

import "github.com/bwmarrin/discordgo"

func HandleJoinGame(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if gameSession == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "There is no game currently running. Start one with `!startgame`.",
			},
		})
		return
	}

}
