package discord

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/pkg/entities"
	"github.com/fadedpez/tucoramirez/pkg/services/blackjack"
)

func (b *Bot) handleReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Bot is ready: %v#%v", s.State.User.Username, s.State.User.Discriminator)

	// Signal that the bot is ready
	select {
	case b.readyChan <- struct{}{}:
		log.Printf("Sent ready signal")
	default:
		log.Printf("Ready channel already signaled or not being listened to")
	}

	// Register commands
	log.Printf("Registering slash commands...")

	// Register the blackjack command
	command := &discordgo.ApplicationCommand{
		Name:        "blackjack",
		Description: "¡Juega blackjack conmigo, amigo! Start a new game of blackjack",
	}

	_, err := s.ApplicationCommandCreate(s.State.User.ID, "", command)
	if err != nil {
		log.Printf("Error creating command %v: %v", command.Name, err)
	} else {
		log.Printf("Successfully registered command: %v", command.Name)
	}

	// Register the wallet command
	walletCommand := &discordgo.ApplicationCommand{
		Name:        "wallet",
		Description: "Check your wallet balance, take loans, or pay off loans",
	}

	_, err = s.ApplicationCommandCreate(s.State.User.ID, "", walletCommand)
	if err != nil {
		log.Printf("Error creating command %v: %v", walletCommand.Name, err)
	} else {
		log.Printf("Successfully registered command: %v", walletCommand.Name)
	}

	log.Printf("Finished registering slash commands")
}

func (b *Bot) handleInteractions(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Printf("Received interaction type: %v", i.Type)
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
		} else if i.ApplicationCommandData().Name == "wallet" {
			log.Printf("Routing to wallet command handler")
			b.handleWalletCommand(s, i)
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

	case "wallet_loan":
		b.handleWalletLoan(s, i)

	case "wallet_repay":
		b.handleWalletRepayment(s, i)

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

	// Check if player is already in the lobby
	if _, alreadyJoined := lobby.Players[i.Member.User.ID]; alreadyJoined {
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "¡Tranquilo, amigo! *tips hat* You're already at the table. Just wait for the game to start.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return
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

	// Check if it's the player's turn for hit/stand actions
	if (action == "hit" || action == "stand") && !game.IsPlayerTurn(i.Member.User.ID) {
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Espera tu turno, amigo! *taps cards impatiently* It's not your turn!",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return
	}

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
			} else {
				// All players bust, set game state to complete
				game.State = entities.StateComplete
				log.Printf("All players bust in channel %s, setting game to complete", i.ChannelID)
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

	// Check if all players are done, and if so, make sure game state is complete
	if game.CheckAllPlayersDone() && game.State != entities.StateComplete {
		// If not all players bust, play dealer's turn
		if !game.CheckAllPlayersBust() {
			// Play dealer's turn
			if err := game.PlayDealer(); err != nil {
				log.Printf("Error playing dealer's turn: %v", err)
			}
		}
		// Set game to complete state
		game.State = entities.StateComplete
		log.Printf("All players are done in channel %s, setting game to complete", i.ChannelID)

		// Recreate the embed with the updated game state
		embed = createGameEmbed(game, s, i.GuildID)
	}

	// Determine which components to show based on game state
	var components []discordgo.MessageComponent

	// Check if the game is over
	gameOver := game.State == entities.StateComplete || game.CheckAllPlayersDone()

	// If the game is over, show play again button
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

	// Send a new message with the lobby UI instead of updating the existing one
	content := "¡Bienvenidos! *Tuco shuffles the cards with flair* Who's ready to play?"
	_, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Content:    content,
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		log.Printf("Error sending new lobby message: %v", err)
		return
	}

	// Acknowledge the interaction to avoid the "interaction failed" message
	content = "Starting a new game..."
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
	if err != nil {
		log.Printf("Error acknowledging interaction: %v", err)
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

func (b *Bot) handleWalletCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Printf("Handling wallet command for channel: %s", i.ChannelID)

	// Acknowledge the interaction immediately to prevent timeout
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Error acknowledging interaction: %v", err)
		return
	}

	// Get the user's wallet
	userWallet, _, err := b.walletService.GetOrCreateWallet(context.Background(), i.Member.User.ID)
	if err != nil {
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "¡Ay, caramba! *looks confused* Failed to retrieve your wallet!",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return
	}

	// Create wallet embed
	balanceStr := fmt.Sprintf("$%d", userWallet.Balance)

	fields := []*discordgo.MessageEmbedField{
		{
			Name:  "Balance",
			Value: balanceStr,
		},
	}

	if userWallet.LoanAmount > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Loan Amount",
			Value: fmt.Sprintf("$%d", userWallet.LoanAmount),
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Your Wallet",
		Description: fmt.Sprintf("Here's your current wallet status, %s", i.Member.User.Username),
		Color:       0x00FF00, // Green color
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Last Updated: %s", userWallet.LastUpdated.Format(time.RFC1123)),
		},
	}

	// Create components for wallet actions
	components := []discordgo.MessageComponent{}

	// Add action buttons
	actionRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Get Loan",
				Style:    discordgo.PrimaryButton,
				CustomID: "wallet_loan",
			},
		},
	}

	// Only show repay button if there's a loan to repay
	if userWallet.LoanAmount > 0 {
		actionRow.Components = append(actionRow.Components, discordgo.Button{
			Label:    "Repay Loan",
			Style:    discordgo.SuccessButton,
			CustomID: "wallet_repay",
		})
	}

	// Add the action row if it has any components
	if len(actionRow.Components) > 0 {
		components = append(components, actionRow)
	}

	// Send followup message with wallet info
	_, err = s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Content:    "Your wallet:",
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		log.Printf("Error sending followup message: %v", err)
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

func (b *Bot) handleWalletLoan(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID
	ctx := context.Background()

	// Give the user a loan of 100
	loanAmount := int64(100)
	err := b.walletService.TakeLoan(ctx, userID, loanAmount)
	if err != nil {
		log.Printf("Error adding loan: %v", err)
		content := "¡Ay caramba! *looks worried* Something went wrong with the loan. Try again later!"
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return
	}

	// Update the wallet display
	updatedWallet, _, err := b.walletService.GetOrCreateWallet(ctx, userID)
	if err != nil {
		log.Printf("Error getting updated wallet: %v", err)
		return
	}

	// Update the message with the new wallet info
	b.updateWalletMessage(s, i, updatedWallet)
}

func (b *Bot) handleWalletRepayment(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID
	ctx := context.Background()

	// Get the user's wallet
	wallet, _, err := b.walletService.GetOrCreateWallet(ctx, userID)
	if err != nil {
		log.Printf("Error getting wallet: %v", err)
		return
	}

	if wallet.LoanAmount <= 0 {
		content := "¡No hay deuda, amigo! *looks confused* You don't have any loans to repay!"
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return
	}

	// Check if user has enough balance to repay
	if wallet.Balance < wallet.LoanAmount {
		content := "¡No tienes suficiente dinero! *counts your chips* You don't have enough to repay your loan!"
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return
	}

	// Repay the loan - use the full loan amount
	err = b.walletService.RepayLoan(ctx, userID, wallet.LoanAmount)
	if err != nil {
		log.Printf("Error repaying loan: %v", err)
		content := "¡Ay caramba! *looks worried* Something went wrong with the repayment. Try again later!"
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return
	}

	// Update the wallet display
	updatedWallet, _, err := b.walletService.GetOrCreateWallet(ctx, userID)
	if err != nil {
		log.Printf("Error getting updated wallet: %v", err)
		return
	}

	// Update the message with the new wallet info
	b.updateWalletMessage(s, i, updatedWallet)
}

func (b *Bot) updateWalletMessage(s *discordgo.Session, i *discordgo.InteractionCreate, userWallet *entities.Wallet) {
	// Format balance with dollar sign
	balanceStr := fmt.Sprintf("$%d", userWallet.Balance)

	// Create embed fields
	fields := []*discordgo.MessageEmbedField{
		{
			Name:  "Balance",
			Value: balanceStr,
		},
	}

	// Only show loan amount if it exists
	if userWallet.LoanAmount > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Loan Amount",
			Value: fmt.Sprintf("$%d", userWallet.LoanAmount),
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Your Wallet",
		Description: fmt.Sprintf("Here's your current wallet status, %s", i.Member.User.Username),
		Color:       0x00FF00, // Green color
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Last Updated: %s", userWallet.LastUpdated.Format(time.RFC1123)),
		},
	}

	// Create components for wallet actions
	components := []discordgo.MessageComponent{}

	// Add action buttons
	actionRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Get Loan",
				Style:    discordgo.PrimaryButton,
				CustomID: "wallet_loan",
			},
		},
	}

	// Only show repay button if there's a loan to repay
	if userWallet.LoanAmount > 0 {
		actionRow.Components = append(actionRow.Components, discordgo.Button{
			Label:    "Repay Loan",
			Style:    discordgo.SuccessButton,
			CustomID: "wallet_repay",
		})
	}

	// Add the action row if it has any components
	if len(actionRow.Components) > 0 {
		components = append(components, actionRow)
	}

	// Edit the interaction response
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content:    &i.Message.Content,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})

	if err != nil {
		log.Printf("Error updating wallet message: %v", err)
	}
}
