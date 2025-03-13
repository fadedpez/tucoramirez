package discord

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/pkg/entities"
	"github.com/fadedpez/tucoramirez/pkg/services/blackjack"
)

func (b *Bot) handleReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Bot is ready: %v#%v", s.State.User.Username, s.State.User.Discriminator)

	// Register the blackjack command
	command := &discordgo.ApplicationCommand{
		Name:        "blackjack",
		Description: "¡Juega blackjack conmigo, amigo! Start a new game of blackjack",
	}

	_, err := s.ApplicationCommandCreate(s.State.User.ID, "", command)
	if err != nil {
		log.Printf("Error creating command %v: %v", command.Name, err)
	}
}

func (b *Bot) handleInteractions(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Printf("Received interaction type: %v", i.Type)

	// Check if we've already processed this interaction
	b.interactionMu.RLock()
	if _, processed := b.processedInteractions[i.ID]; processed {
		b.interactionMu.RUnlock()
		log.Printf("Skipping already processed interaction: %s", i.ID)
		return
	}
	b.interactionMu.RUnlock()

	// Mark as processed
	b.interactionMu.Lock()
	b.processedInteractions[i.ID] = true

	// Periodically clean up old interactions (every 100 interactions or so)
	if len(b.processedInteractions) > 100 && time.Since(b.lastCleanupTime) > 5*time.Minute {
		log.Printf("Cleaning up processed interactions map, current size: %d", len(b.processedInteractions))
		// Only keep interactions from the last 10 minutes
		for id := range b.processedInteractions {
			delete(b.processedInteractions, id)
		}
		b.lastCleanupTime = time.Now()
		log.Printf("Cleaned up processed interactions map, new size: %d", len(b.processedInteractions))
	}
	b.interactionMu.Unlock()

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		log.Printf("Received application command: %s", i.ApplicationCommandData().Name)
		if i.ApplicationCommandData().Name == "blackjack" {
			log.Printf("Routing to blackjack command handler")
			b.handleBlackjackCommand(s, i)
		}

	case discordgo.InteractionMessageComponent:
		log.Printf("Received message component interaction: %s", i.MessageComponentData().CustomID)
		b.handleMessageComponentInteraction(s, i)
	}
}

func (b *Bot) handleMessageComponentInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Printf("Received message component interaction: %s", i.MessageComponentData().CustomID)

	// Acknowledge the interaction immediately to prevent timeout
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		log.Printf("Error acknowledging component interaction: %v", err)
		return
	}

	switch i.MessageComponentData().CustomID {
	case "join_game":
		b.handleJoinGame(s, i)

	case "start_game":
		b.handleStartGame(s, i)

	case "hit", "stand":
		b.handleGameAction(s, i)

	case "play_again":
		b.handlePlayAgain(s, i)

	default:
		log.Printf("Unknown component ID: %s", i.MessageComponentData().CustomID)
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "¡Ay caramba! *looks confused* I don't know what to do with that button!",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
	}
}

func (b *Bot) handleJoinGame(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Check if there's a lobby in this channel
	b.mu.RLock()
	lobby, lobbyExists := b.lobbies[i.ChannelID]
	game, gameExists := b.games[i.ChannelID]
	b.mu.RUnlock()

	if !lobbyExists {
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "¡Ay caramba! *frantically searches the casino* No lobby found in this channel! Start a new game with /blackjack",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return
	}

	if gameExists {
		if _, playing := game.Players[i.Member.User.ID]; playing {
			_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "¡Espera un momento! *counts his chips nervously* You're already playing in a game! Finish that one first, ¿eh?",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			if err != nil {
				log.Printf("Error sending followup message: %v", err)
			}
			return
		}
	}

	// Add player to lobby
	b.mu.Lock()
	lobby.Players[i.Member.User.ID] = true
	b.mu.Unlock()

	// Update lobby display
	b.updateLobbyDisplay(s, i, lobby)
}

func (b *Bot) handleStartGame(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Printf("Handling start game for channel: %s", i.ChannelID)

	// Check if there's a lobby in this channel
	b.mu.RLock()
	lobby, exists := b.lobbies[i.ChannelID]
	log.Printf("Lobby exists for channel %s: %v (lobby map size: %d)", i.ChannelID, exists, len(b.lobbies))
	b.mu.RUnlock()

	if !exists {
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "¡Ay caramba! *frantically searches the casino* No lobby found in this channel! Start a new game with /blackjack",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return
	}

	// Verify sender is owner
	if i.Member.User.ID != lobby.OwnerID {
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "¡OIGA! *adjusts golden rings menacingly* Only El Jefe can start the game! ¿Comprende?",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return
	}

	// Create new game
	game := blackjack.NewGame(i.ChannelID, b.repo)

	// Add all lobby players to game
	for playerID := range lobby.Players {
		if err := game.AddPlayer(playerID); err != nil {
			_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "¡Ay, caramba! *drops cards everywhere* Failed to add players to the game!",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			if err != nil {
				log.Printf("Error sending followup message: %v", err)
			}
			return
		}
	}

	// Start the game
	if err := game.Start(); err != nil {
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "¡Madre de Dios! *shuffles cards nervously* Failed to start the game!",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return
	}

	// Store game in memory
	b.mu.Lock()
	b.games[i.ChannelID] = game
	b.mu.Unlock()

	// Create game state embed
	embed := createGameEmbed(game, s, i.GuildID)

	// Update the message with game UI and embed
	content := "¡Vamos a jugar! *Tuco deals the cards with a flourish*"
	_, err := s.FollowupMessageEdit(i.Interaction, i.Message.ID, &discordgo.WebhookEdit{
		Content: &content,
		Embeds:  &[]*discordgo.MessageEmbed{embed},
		Components: &[]discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Hit",
						Style:    discordgo.PrimaryButton,
						CustomID: "hit",
					},
					discordgo.Button{
						Label:    "Stand",
						Style:    discordgo.DangerButton,
						CustomID: "stand",
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Error updating message with game UI: %v", err)
		return
	}

	// Now that we've successfully updated the UI, we can safely delete the lobby
	b.mu.Lock()
	delete(b.lobbies, i.ChannelID)
	b.mu.Unlock()
}

func (b *Bot) handleGameAction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	action := i.MessageComponentData().CustomID
	log.Printf("Handling game action: %s for channel: %s", action, i.ChannelID)

	// Check if there's a game in this channel
	b.mu.RLock()
	game, exists := b.games[i.ChannelID]
	b.mu.RUnlock()

	if !exists {
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "¡Ay caramba! *looks around confused* No game found in this channel! Start a new game with /blackjack",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return
	}

	// Process the action
	var err error

	switch action {
	case "hit":
		err = game.Hit(i.Member.User.ID)
		// Check if the error is a bust, which is actually a valid game state
		if err == blackjack.ErrHandBust {
			log.Printf("Player %s busted! This is a valid game state, not an error.", i.Member.User.ID)
			err = nil // Clear the error since bust is a valid game state
		}
	case "stand":
		err = game.Stand(i.Member.User.ID)

		// After stand, check if all players are done and play dealer if needed
		if err == nil && game.CheckAllPlayersDone() {
			if !game.CheckAllPlayersBust() {
				// Play dealer's turn if not all players bust
				err = game.PlayDealer()
			}
		}
	}

	// Only treat non-bust errors as actual errors
	if err != nil && err != blackjack.ErrHandBust {
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("¡Ay, no bueno! *shuffles cards nervously* Something went wrong: %v", err),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return
	}

	// Create updated game state embed
	embed := createGameEmbed(game, s, i.GuildID)

	// Update the message with new game state
	content := "¡Vamos a jugar! *Tuco deals the cards with a flourish*"

	// Determine which components to show based on game state
	var components []discordgo.MessageComponent

	// Check if all players are done to determine if the game is over
	gameOver := game.CheckAllPlayersDone()

	if gameOver {
		// Game is over, show play again button
		components = []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Play Again",
						Style:    discordgo.SuccessButton,
						CustomID: "play_again",
					},
				},
			},
		}

		// Update content to show game is over
		content = "¡El juego ha terminado! *Tuco counts the chips with a grin*"
	} else {
		// Game is still in progress, show hit/stand buttons
		components = []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Hit",
						Style:    discordgo.PrimaryButton,
						CustomID: "hit",
					},
					discordgo.Button{
						Label:    "Stand",
						Style:    discordgo.DangerButton,
						CustomID: "stand",
					},
				},
			},
		}
	}

	// Update the message
	_, err = s.FollowupMessageEdit(i.Interaction, i.Message.ID, &discordgo.WebhookEdit{
		Content:    &content,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		log.Printf("Error updating message with game state: %v", err)
		return
	}

	// If the game is over, clean up
	if gameOver {
		// If the game state is still playing but all players are done, update to complete
		if game.State == entities.StatePlaying || game.State == entities.StateDealer {
			game.State = entities.StateComplete
			log.Printf("Game in channel %s is complete, updating state", i.ChannelID)
		}

		// Send a game completion image if available
		if b.imageService != nil {
			image := b.imageService.GetRandomImage()
			if image != nil && image.URL != "" {
				// Send a separate message with the image
				imageContent := ""
				_, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
					Content: imageContent,
					Embeds: []*discordgo.MessageEmbed{
						{
							Image: &discordgo.MessageEmbedImage{
								URL: image.URL,
							},
						},
					},
				})
				if err != nil {
					log.Printf("Error sending game completion image: %v", err)
				} else {
					log.Printf("Sent game completion image for channel %s", i.ChannelID)
				}
			}
		}

		b.mu.Lock()
		delete(b.games, i.ChannelID)
		b.mu.Unlock()
	}
}

func (b *Bot) handlePlayAgain(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Printf("Handling play again for channel: %s", i.ChannelID)

	// Create a new lobby with the same owner
	lobby := &GameLobby{
		OwnerID: i.Member.User.ID,
		Players: make(map[string]bool),
	}

	// Add the owner as the first player
	lobby.Players[i.Member.User.ID] = true

	// Store the lobby
	b.mu.Lock()
	b.lobbies[i.ChannelID] = lobby
	log.Printf("Created and stored new lobby for channel %s, owner: %s, lobby map size: %d", i.ChannelID, i.Member.User.ID, len(b.lobbies))
	b.mu.Unlock()

	// Create lobby embed and buttons
	embed := createLobbyEmbed(lobby)
	components := createLobbyButtons(lobby.OwnerID)

	// Update the message with lobby UI
	content := "\u00a1Bienvenidos! *Tuco shuffles the cards with flair* Who's ready to play?"
	_, err := s.FollowupMessageEdit(i.Interaction, i.Message.ID, &discordgo.WebhookEdit{
		Content:    &content,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		log.Printf("Error updating message with lobby UI: %v", err)
		return
	}
}

func (b *Bot) updateLobbyDisplay(s *discordgo.Session, i *discordgo.InteractionCreate, lobby *GameLobby) {
	// Create updated lobby embed
	embed := createLobbyEmbed(lobby)
	components := createLobbyButtons(lobby.OwnerID)

	// Update the message
	_, err := s.FollowupMessageEdit(i.Interaction, i.Message.ID, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		log.Printf("Error updating lobby display: %v", err)
	}
}

func (b *Bot) sendGameState(s *discordgo.Session, i *discordgo.InteractionCreate, game *blackjack.Game) {
	// Create game state message
	embed := createGameEmbed(game, s, i.GuildID)

	// Send as a followup message
	_, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Content: "Current game state:",
		Embeds:  []*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		log.Printf("Error sending game state: %v", err)

		// Try sending a regular channel message as fallback
		_, msgErr := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
			Content: "Current game state:",
			Embeds:  []*discordgo.MessageEmbed{embed},
		})
		if msgErr != nil {
			log.Printf("Failed to send fallback game state message: %v", msgErr)
		}
	}
}

func (b *Bot) handleBlackjackCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Printf("Handling blackjack command for channel: %s", i.ChannelID)

	// Check if there's already a game or lobby in this channel before proceeding
	b.mu.RLock()
	gameExists := false
	lobbyExists := false
	_, gameExists = b.games[i.ChannelID]
	_, lobbyExists = b.lobbies[i.ChannelID]
	log.Printf("Channel %s - Game exists: %v, Lobby exists: %v", i.ChannelID, gameExists, lobbyExists)
	b.mu.RUnlock()

	// If there's already a lobby or game in this channel, don't create a new one
	if lobbyExists || gameExists {
		// Just acknowledge the interaction to prevent timeout
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Printf("Error acknowledging interaction: %v", err)
		}

		// Send an ephemeral message to the user
		_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "A game or lobby already exists in this channel!",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error responding with lobby: %v", err)
		}
		return
	}

	// Acknowledge the interaction immediately to prevent timeout
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("Error acknowledging interaction: %v", err)
		return
	}

	// Force cleanup any stale games or lobbies in this channel
	b.mu.Lock()
	if _, exists := b.games[i.ChannelID]; exists {
		log.Printf("Cleaning up stale game in channel %s", i.ChannelID)
		delete(b.games, i.ChannelID)
	}
	if _, exists := b.lobbies[i.ChannelID]; exists {
		log.Printf("Cleaning up stale lobby in channel %s", i.ChannelID)
		delete(b.lobbies, i.ChannelID)
	}
	b.mu.Unlock()

	// Create new lobby
	lobby := &GameLobby{
		OwnerID: i.Member.User.ID,
		Players: make(map[string]bool),
	}
	lobby.Players[i.Member.User.ID] = true

	// Store the lobby
	b.mu.Lock()
	b.lobbies[i.ChannelID] = lobby
	log.Printf("Created and stored new lobby for channel %s, owner: %s, lobby map size: %d", i.ChannelID, i.Member.User.ID, len(b.lobbies))
	b.mu.Unlock()

	// Create lobby embed and buttons
	embed := createLobbyEmbed(lobby)
	components := createLobbyButtons(lobby.OwnerID)

	// Send followup message with lobby info
	_, err = s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Content:    "\u00a1Bienvenidos! *Tuco shuffles the cards with flair* Who's ready to play?",
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		log.Printf("Error sending followup message: %v", err)

		// Try sending a regular channel message as fallback
		msg, msgErr := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
			Content:    "\u00a1Bienvenidos! *Tuco shuffles the cards with flair* Who's ready to play?",
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		})

		if msgErr != nil {
			log.Printf("Failed to send fallback message: %v", msgErr)
			b.mu.Lock()
			delete(b.lobbies, i.ChannelID)
			b.mu.Unlock()
		} else {
			log.Printf("Successfully sent fallback message for lobby in channel %s: %s", i.ChannelID, msg.ID)
		}
	}
}

func (b *Bot) displayGameState(s *discordgo.Session, i *discordgo.InteractionCreate, gameState interface{}) {
	log.Printf("Displaying game state")
	var responseType discordgo.InteractionResponseType
	if i.Type == discordgo.InteractionApplicationCommand {
		responseType = discordgo.InteractionResponseChannelMessageWithSource
	} else {
		responseType = discordgo.InteractionResponseUpdateMessage
	}

	switch gameState := gameState.(type) {
	case *GameLobby:
		embed := createLobbyEmbed(gameState)
		components := createLobbyButtons(gameState.OwnerID)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: responseType,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: components,
			},
		})
	case *blackjack.Game:
		// Check if deck was shuffled
		if gameState.WasShuffled() {
			// Send a message about shuffling
			s.ChannelMessageSend(i.ChannelID, "*We've been playing a long time eh my friends? Let Tuco shuffle the deck, maybe it bring Tuco more luck.* ")
		}

		// Create the game state embed
		embed := createGameEmbed(gameState, s, i.GuildID)
		components := createGameButtons(gameState)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: responseType,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: components,
			},
		})
	}
}
