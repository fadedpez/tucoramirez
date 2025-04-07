package discord

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
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
	log.Printf("Handling component interaction: %s for user %s",
		i.MessageComponentData().CustomID, i.Member.User.ID)

	// Acknowledge the interaction immediately
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		log.Printf("Error acknowledging interaction: %v", err)
		return
	}

	// Handle different component types
	customID := i.MessageComponentData().CustomID
	switch {
	case customID == "join_game":
		b.handleJoinGame(s, i)

	case customID == "start_game":
		b.handleStartGame(s, i)

	case customID == "hit" || customID == "stand":
		b.handleGameAction(s, i)

	case customID == "play_again":
		b.handlePlayAgain(s, i)

	case customID == "wallet_loan":
		b.handleWalletLoan(s, i)

	case customID == "wallet_repay":
		b.handleWalletRepayment(s, i)

	case strings.HasPrefix(customID, "bet_"):
		betAmount, err := strconv.ParseInt(strings.TrimPrefix(customID, "bet_"), 10, 64)
		if err != nil {
			log.Printf("Error parsing bet amount: %v", err)
			_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "*Tuco looks confused* ¿Qué? I don't understand that command.",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			if err != nil {
				log.Printf("Error sending followup message: %v", err)
			}
			return
		}
		b.handleBet(s, i, betAmount)

	default:
		log.Printf("Unknown component ID: %s", customID)
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "*Tuco looks confused* ¿Qué? I don't understand that command.",
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
				Content: "¡Espera un momento! *taps cards impatiently* You're already playing in a game! Finish that one first, ¿eh?",
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
	// Check if there's already a lobby in this channel before proceeding
	b.mu.RLock()
	lobby, exists := b.lobbies[i.ChannelID]
	b.mu.RUnlock()

	if !exists {
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "¡Ay caramba! *looks around confused* No lobby found in this channel! Create one with /lobby first.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return
	}

	// Verify the sender is the lobby owner
	if lobby.OwnerID != i.Member.User.ID {
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "¡No, no, no! *wags finger* Only the lobby owner can start the game!",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return
	}

	// Create a new game
	game := blackjack.NewGame(i.ChannelID, b.repo)

	// Add all players from the lobby
	for playerID := range lobby.Players {
		if err := game.AddPlayer(playerID); err != nil {
			log.Printf("Error adding player %s to game: %v", playerID, err)
		}
	}

	// Start the game (this will transition to betting phase)
	if err := game.Start(); err != nil {
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("¡Ay, no bueno! *shuffles cards nervously* Something went wrong: %v", err),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return
	}

	// Store the game in the bot's state
	b.mu.Lock()
	b.games[i.ChannelID] = game
	// Remove the lobby since the game has started
	delete(b.lobbies, i.ChannelID)
	b.mu.Unlock()

	// Create betting buttons for all players
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Bet $5",
					Style:    discordgo.SuccessButton,
					CustomID: "bet_5",
				},
				discordgo.Button{
					Label:    "Bet $10",
					Style:    discordgo.SuccessButton,
					CustomID: "bet_10",
				},
				discordgo.Button{
					Label:    "Bet $25",
					Style:    discordgo.SuccessButton,
					CustomID: "bet_25",
				},
			},
		},
	}

	// Update the existing message with the betting UI
	content := "¡Vamos a jugar! *Tuco shuffles the cards with flair* Place your bets to begin!"
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content:    &content,
		Embeds:     &[]*discordgo.MessageEmbed{createGameEmbed(game, s, i.GuildID)},
		Components: &components,
	})
	if err != nil {
		log.Printf("Error updating message with betting UI: %v", err)
		return
	}

	// Update the betting UI with detailed player information
	b.updateBettingUI(s, i, game)

	log.Printf("Started new game in channel %s with %d players", i.ChannelID, len(game.Players))
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
	case "stand":
		err = game.Stand(i.Member.User.ID)
	}

	// Only treat non-bust errors as actual errors
	if err != nil {
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("¡Ay, no bueno! *shuffles cards nervously* Something went wrong: %v", err),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return
	}

	// Use the service method to check if game is complete and handle dealer play/payouts
	ctx := context.Background()
	gameComplete, err := game.CompleteGameIfDone(ctx, b.walletService)
	if err != nil {
		log.Printf("Error completing game: %v", err)
	}

	// Create updated game state embed
	embed := b.displayGameState(s, i, game)

	// Determine which components to show based on game state
	var components []discordgo.MessageComponent
	var content string

	// If the game is over, show play again button
	if gameComplete {
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

		// Content for in-progress game
		content = "¡Vamos a jugar! *Tuco waits for players to make their moves*"
	}

	// Update the message
	var err2 error
	_, err2 = s.FollowupMessageEdit(i.Interaction, i.Message.ID, &discordgo.WebhookEdit{
		Content:    &content,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err2 != nil {
		log.Printf("Error updating message with game state: %v", err2)
		return
	}

	// If the game is over, clean up
	if gameComplete {
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
	embed := b.displayGameState(s, i, game)

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
	// Acknowledge the interaction immediately to prevent timeout
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
		return
	}

	// Get the user's wallet
	userWallet, _, err := b.walletService.GetOrCreateWallet(context.Background(), i.Member.User.ID)
	if err != nil {
		b.sendWalletErrorResponse(s, i, err, "Error retrieving your wallet. Please try again later.")
		return
	}

	// Update the wallet message - use false for isUpdate since this is a new message
	b.updateWalletMessage(s, i, userWallet, false)
}

func (b *Bot) displayGameState(s *discordgo.Session, i *discordgo.InteractionCreate, game interface{}) *discordgo.MessageEmbed {
	log.Printf("Displaying game state")
	var responseType discordgo.InteractionResponseType
	if i.Type == discordgo.InteractionApplicationCommand {
		responseType = discordgo.InteractionResponseChannelMessageWithSource
	} else {
		responseType = discordgo.InteractionResponseUpdateMessage
	}

	switch gameState := game.(type) {
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
		return embed
	case *blackjack.Game:
		// Check if deck was shuffled using service method
		wasShuffled, shuffleMessage := gameState.GetShuffleInfo()
		if wasShuffled {
			// Send a message about shuffling
			s.ChannelMessageSend(i.ChannelID, shuffleMessage)
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
		return embed
	}
	return &discordgo.MessageEmbed{}
}

func (b *Bot) handleBet(s *discordgo.Session, i *discordgo.InteractionCreate, betAmount int64) error {
	// Get the game
	b.mu.RLock()
	game, exists := b.games[i.ChannelID]
	b.mu.RUnlock()

	if !exists {
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "¡Ay caramba! *looks around confused* No game found in this channel!",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return fmt.Errorf("no game found in channel %s", i.ChannelID)
	}

	// Use the service method to validate the bet
	if err := game.ValidateBet(i.Member.User.ID); err != nil {
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("¡No es posible! *shakes head* %v", err),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return fmt.Errorf("bet validation failed: %w", err)
	}

	// Use the service method to place bet and update wallet
	ctx := context.Background()
	loanGiven, err := game.PlaceBetWithWalletUpdate(ctx, i.Member.User.ID, betAmount, b.walletService)
	if err != nil {
		log.Printf("Error placing bet for player %s: %v", i.Member.User.ID, err)
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "¡Algo salió mal! *scratches head* Could not place your bet!",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending followup message: %v", err)
		}
		return fmt.Errorf("error placing bet: %v", err)
	}

	// If a loan was given, notify the player
	if loanGiven {
		loanAmount := b.walletService.GetStandardLoanIncrement()
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("*Tuco smiles and slides some chips your way* ¡No problemo, amigo! I've given you a loan of $%d. Don't forget to pay me back... or else! *winks*", loanAmount),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			log.Printf("Error sending loan message: %v", err)
		}
	}

	// Notify player of successful bet
	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: fmt.Sprintf("*Tuco nods approvingly* ¡Buena apuesta, amigo! You bet $%d", betAmount),
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		log.Printf("Error sending bet confirmation: %v", err)
	}

	// Check if all players have placed bets and game has transitioned to DEALING or PLAYING
	if game.State == entities.StateDealing || game.State == entities.StatePlaying {
		// Update the game UI with the playing state
		return b.updateGameUI(s, i, game)
	} else {
		// Otherwise, update the betting UI to show the next player's turn
		return b.updateBettingUI(s, i, game)
	}
}

func (b *Bot) updateBettingUI(s *discordgo.Session, i *discordgo.InteractionCreate, game *blackjack.Game) error {
	log.Printf("Updating betting UI")

	// Get player information from the service layer
	ctx := context.Background()
	playersInfo, err := game.GetAllPlayersInfo(ctx, b.walletService)
	if err != nil {
		log.Printf("Error getting player information: %v", err)
	}

	// Find the player with the highest balance
	var highestBalance int64
	var richestPlayerID string
	for _, playerInfo := range playersInfo {
		if playerInfo.WalletBalance > highestBalance {
			highestBalance = playerInfo.WalletBalance
			richestPlayerID = playerInfo.PlayerID
		}
	}

	// Get current betting player info
	currentPlayerInfo, err := game.GetCurrentBettingPlayerInfo(ctx, b.walletService)
	if err != nil {
		log.Printf("Error getting current betting player info: %v", err)
		return err
	}

	// Create the embed
	embed := &discordgo.MessageEmbed{
		Title:       "Blackjack - Betting Phase",
		Description: "Place your bets!",
		Color:       0x00FF00, // Green
		Fields:      []*discordgo.MessageEmbedField{},
	}

	// Add fields for each player
	for _, playerInfo := range playersInfo {
		// Get the Discord username
		user, err := s.User(playerInfo.PlayerID)
		if err != nil {
			log.Printf("Error getting user %s: %v", playerInfo.PlayerID, err)
			continue
		}

		// Format the player's status
		statusText := fmt.Sprintf("Balance: $%d", playerInfo.WalletBalance)
		if playerInfo.HasBet {
			statusText += fmt.Sprintf("\nBet: $%d", playerInfo.BetAmount)
		} else {
			statusText += "\nNo bet placed yet"
		}

		// Highlight the current player
		fieldName := user.Username
		if playerInfo.IsCurrentTurn {
			fieldName += " 👈 YOUR TURN"
		}

		// Add crown emoji to player with highest balance
		if playerInfo.PlayerID == richestPlayerID && highestBalance > 0 {
			fieldName += " 👑"
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fieldName,
			Value:  statusText,
			Inline: true,
		})
	}

	// Create bet buttons
	var components []discordgo.MessageComponent

	// Only show bet buttons if it's someone's turn
	if currentPlayerInfo != nil {
		// Get the current player's user object
		currentUser, err := s.User(currentPlayerInfo.PlayerID)
		if err == nil {
			// Add a message showing whose turn it is
			embed.Description = fmt.Sprintf("It's %s's turn to bet!", currentUser.Username)
		}

		// Only show bet buttons to the current player
		if i.Member != nil && i.Member.User != nil && i.Member.User.ID == currentPlayerInfo.PlayerID {
			// Fixed bet options: $5, $10, and $25 only
			betOptions := []int64{5, 10, 25}

			// Create bet buttons
			betButtons := []discordgo.MessageComponent{}
			for _, amount := range betOptions {
				// Only show bet options the player can afford
				if amount <= currentPlayerInfo.WalletBalance {
					betButtons = append(betButtons, discordgo.Button{
						Label:    fmt.Sprintf("$%d", amount),
						Style:    discordgo.SuccessButton,
						CustomID: fmt.Sprintf("bet_%d", amount),
					})
				}
			}

			// Split buttons into rows of 5 max
			for i := 0; i < len(betButtons); i += 5 {
				end := i + 5
				if end > len(betButtons) {
					end = len(betButtons)
				}

				components = append(components, discordgo.ActionsRow{
					Components: betButtons[i:end],
				})
			}
		}
	} else {
		// No current player - game might be waiting to start or already in progress
		embed.Description = "Waiting for players to join..."
	}

	// Edit the message with the updated UI
	var msgErr error
	_, msgErr = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Embed:      embed,
		ID:         i.Message.ID,
		Channel:    i.ChannelID,
		Components: &components,
	})

	return msgErr
}

func (b *Bot) updateGameUI(s *discordgo.Session, i *discordgo.InteractionCreate, game *blackjack.Game) error {
	log.Printf("Updating game UI - Game state: %s, PayoutsProcessed: %v", game.State, game.PayoutsProcessed)

	// Create the game state embed
	embed := b.displayGameState(s, i, game)

	// Create game buttons based on game state
	components := createGameButtons(game)

	// Update the message with appropriate content based on game state
	var content string
	switch game.State {
	case entities.StateDealing:
		content = "¡Vamos a jugar! *Tuco deals the cards with a flourish*"
	case entities.StatePlaying:
		// Get the current player's name whose turn it is to play
		if len(game.PlayerOrder) > 0 && game.CurrentTurn < len(game.PlayerOrder) {
			currentPlayerID := game.PlayerOrder[game.CurrentTurn]
			currentPlayer, err := s.User(currentPlayerID)
			if err != nil {
				log.Printf("Error getting user %s: %v", currentPlayerID, err)
				content = "¡Vamos a jugar! *Tuco waits for players to make their moves*"
			} else {
				// Get player's wallet
				wallet, _, err := b.walletService.GetOrCreateWallet(context.Background(), currentPlayerID)
				walletInfo := fmt.Sprintf(" ($%d)", wallet.Balance)
				if err != nil {
					log.Printf("Error getting wallet for player %s: %v", currentPlayerID, err)
					walletInfo = ""
				}
				content = fmt.Sprintf("¡Vamos a jugar! *Tuco looks at %s%s* It's your turn to play!", currentPlayer.Username, walletInfo)
			}
		} else {
			content = "¡Vamos a jugar! *Tuco waits for players to make their moves*"
		}
	case entities.StateComplete:
		content = "¡El juego ha terminado! *Tuco counts the chips with a grin*"

		// Process payouts for all players if they haven't been processed yet
		if !game.PayoutsProcessed {
			log.Printf("Processing payouts for completed game in channel %s", i.ChannelID)
			ctx := context.Background()
			if err := game.FinishGame(ctx, b.walletService); err != nil {
				log.Printf("Error finishing game: %v", err)
			}
		}
	default:
		content = "¡Vamos a jugar! *Tuco shuffles the cards*"
	}

	// Edit the original message
	var err2 error
	_, err2 = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content:    &content,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err2 != nil {
		log.Printf("Error updating game UI: %v", err2)
		return fmt.Errorf("error updating game UI: %v", err2)
	}

	return nil
}

func (b *Bot) handleWalletLoan(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Printf("Processing loan request for user %s", i.Member.User.ID)

	// The interaction is already acknowledged in handleMessageComponentInteraction
	// so we don't need to acknowledge it again here

	// Get the standard loan increment from the service
	loanAmount := b.walletService.GetStandardLoanIncrement()

	// Validate loan eligibility
	err := b.walletService.ValidateLoan(context.Background(), i.Member.User.ID, loanAmount)
	if err != nil {
		b.sendWalletErrorResponse(s, i, err, fmt.Sprintf("Cannot take a loan: %v", err))
		return
	}

	// Process the loan
	updatedWallet, _, err := b.walletService.GiveLoan(context.Background(), i.Member.User.ID, loanAmount)
	if err != nil {
		b.sendWalletErrorResponse(s, i, err, "Error processing your loan. Please try again later.")
		return
	}

	// Update the message with the new wallet info
	log.Printf("Loan processed successfully for user %s, updating wallet message", i.Member.User.ID)
	b.updateWalletMessage(s, i, updatedWallet, true)
}

func (b *Bot) handleWalletRepayment(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Printf("Processing loan repayment request for user %s", i.Member.User.ID)

	// The interaction is already acknowledged in handleMessageComponentInteraction
	// so we don't need to acknowledge it again here

	// Get the repayment amount from the service
	repaymentAmount, err := b.walletService.CalculateRepaymentAmount(context.Background(), i.Member.User.ID)
	if err != nil {
		b.sendWalletErrorResponse(s, i, err, fmt.Sprintf("Cannot calculate repayment amount: %v", err))
		return
	}

	// Validate repayment eligibility
	err = b.walletService.ValidateRepayment(context.Background(), i.Member.User.ID, repaymentAmount)
	if err != nil {
		b.sendWalletErrorResponse(s, i, err, fmt.Sprintf("Oye, no one pulls a fast one on Tuco! %v", err))
		return
	}

	// Process the repayment
	err = b.walletService.RepayLoan(context.Background(), i.Member.User.ID, repaymentAmount)
	if err != nil {
		b.sendWalletErrorResponse(s, i, err, "Error processing your loan repayment. Please try again later.")
		return
	}

	// Get the updated wallet
	updatedWallet, _, err := b.walletService.GetOrCreateWallet(context.Background(), i.Member.User.ID)
	if err != nil {
		b.sendWalletErrorResponse(s, i, err, "Error retrieving your updated wallet. Please try again later.")
		return
	}

	log.Printf("Loan repayment processed successfully for user %s, updating wallet message", i.Member.User.ID)
	// Update the message with the new wallet info
	b.updateWalletMessage(s, i, updatedWallet, true)
}

func (b *Bot) updateWalletMessage(s *discordgo.Session, i *discordgo.InteractionCreate, userWallet *entities.Wallet, isUpdate bool) error {
	log.Printf("Updating wallet message - isUpdate: %v, hasMessage: %v, userID: %s",
		isUpdate, i.Message != nil, i.Member.User.ID)

	embed, components := b.createWalletEmbed(userWallet, i.Member.User.ID, i.Member.User.Username)

	if isUpdate && i.Message != nil {
		// Edit the interaction response
		log.Printf("Editing existing wallet message for user %s", i.Member.User.ID)
		_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content:    &i.Message.Content,
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &components,
		})
		if err != nil {
			log.Printf("Error updating wallet message: %v", err)
		}
		return err
	} else {
		// Either isUpdate is false or i.Message is nil
		// Send a new message instead
		if isUpdate {
			log.Printf("Attempted to update wallet message but i.Message was nil, sending new message instead for user %s", i.Member.User.ID)
		} else {
			log.Printf("Sending new wallet message for user %s", i.Member.User.ID)
		}

		_, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content:    "Your wallet:",
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		})
		if err != nil {
			log.Printf("Error sending wallet message: %v", err)
		}
		return err
	}
}

func (b *Bot) createWalletEmbed(userWallet *entities.Wallet, userID string, username string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	// Format balance with dollar sign
	balanceStr := fmt.Sprintf("$%d", userWallet.Balance)

	// Create embed fields
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

	// Get transaction history (ledger)
	transactions, _ := b.walletService.GetRecentTransactions(context.Background(), userID, 10)
	if len(transactions) > 0 {
		transactionsStr := ""
		for _, tx := range transactions {
			// Format with date, amount, and description
			amountStr := fmt.Sprintf("$%+d", tx.Amount)
			dateStr := tx.Timestamp.Format("01/02 15:04")
			transactionsStr += fmt.Sprintf("%s | %s | %s\n", dateStr, amountStr, tx.Description)
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Transaction History",
			Value: transactionsStr,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Your Wallet",
		Description: fmt.Sprintf("Here's your current wallet status, %s", username),
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
				Label:    fmt.Sprintf("Get $%d Loan", b.walletService.GetStandardLoanIncrement()),
				Style:    discordgo.PrimaryButton,
				CustomID: "wallet_loan",
			},
		},
	}

	// Only show repay button if there's a loan to repay
	canRepay, err := b.walletService.CanRepayLoan(context.Background(), userID)
	if err == nil && canRepay {
		// Get the standard repayment amount from the service
		repaymentAmount, _ := b.walletService.CalculateRepaymentAmount(context.Background(), userID)

		actionRow.Components = append(actionRow.Components, discordgo.Button{
			Label:    fmt.Sprintf("Repay $%d", repaymentAmount),
			Style:    discordgo.SuccessButton,
			CustomID: "wallet_repay",
		})
	}

	// Add the action row if it has any components
	if len(actionRow.Components) > 0 {
		components = append(components, actionRow)
	}

	return embed, components
}

// sendWalletErrorResponse sends a standardized error response for wallet operations
func (b *Bot) sendWalletErrorResponse(s *discordgo.Session, i *discordgo.InteractionCreate, err error, message string) {
	log.Printf("Wallet error: %v", err)
	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: message,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		log.Printf("Error sending followup message: %v", err)
	}
}
