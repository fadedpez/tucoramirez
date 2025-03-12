package discord

import (
	"fmt"
	"log"

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
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		if i.ApplicationCommandData().Name == "blackjack" {
			b.handleBlackjackCommand(s, i)
		}

	case discordgo.InteractionMessageComponent:
		switch i.MessageComponentData().CustomID {
		case "join_game":
			// Get the lobby
			channelID := i.ChannelID
			b.mu.RLock()
			lobby, exists := b.lobbies[channelID]
			b.mu.RUnlock()

			if !exists {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "¡Ay caramba! *frantically searches the casino* No lobby found in this channel! Start a new game with /blackjack ",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			// Check if player is already in lobby
			if _, alreadyJoined := lobby.Players[i.Member.User.ID]; alreadyJoined {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "¡AY, DIOS MÍO! *dramatically adjusts golden rings* ¿Qué pasa contigo, amigo? You're ALREADY at my table! ¡Siéntate y espera! Sit down and wait for the game to start! ",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			// Check if there's a game in progress
			b.mu.RLock()
			game, inGame := b.games[channelID]
			b.mu.RUnlock()

			if inGame {
				if _, playing := game.Players[i.Member.User.ID]; playing {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "¡Espera un momento! *counts his chips nervously* You're already playing in a game! Finish that one first, ¿eh? ",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}
			}

			// Add player to lobby
			b.mu.Lock()
			lobby.Players[i.Member.User.ID] = true
			b.mu.Unlock()

			// Update lobby display
			b.displayGameState(s, i, lobby)

		case "start_game":
			channelID := i.ChannelID
			b.mu.RLock()
			lobby, exists := b.lobbies[channelID]
			b.mu.RUnlock()

			if !exists {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "¡Ay caramba! *frantically searches the casino* No lobby found in this channel! Start a new game with /blackjack ",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			// Verify sender is owner
			if i.Member.User.ID != lobby.OwnerID {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "¡OIGA! *adjusts golden rings menacingly* Only El Jefe can start the game! ¿Comprende? ",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			// Create new game
			game := blackjack.NewGame(channelID, b.repo)

			// Add all lobby players to game
			for playerID := range lobby.Players {
				if err := game.AddPlayer(playerID); err != nil {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "¡Ay, caramba! *drops cards everywhere* Failed to add players to the game! ",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}
			}

			// Start the game
			if err := game.Start(); err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "¡Madre de Dios! *shuffles cards nervously* Failed to start the game! ",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			// Store game and clean up lobby
			b.mu.Lock()
			b.games[channelID] = game
			delete(b.lobbies, channelID)
			b.mu.Unlock()

			// Show initial game state
			b.displayGameState(s, i, game)

		case "play_again":
			channelID := i.ChannelID
			b.mu.Lock()
			_, otherGameExists := b.games[channelID]
			_, otherLobbyExists := b.lobbies[channelID]
			if otherGameExists || otherLobbyExists {
				b.mu.Unlock()
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "¡Ay caramba! *adjusts golden rings nervously* There's already another game or lobby in this channel! Wait for it to finish, ¿eh?",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
			delete(b.games, channelID)
			delete(b.lobbies, channelID)
			b.mu.Unlock()

			// Start a new game directly
			b.handleBlackjackCommand(s, i)
			return

		case "hit", "stand":
			channelID := i.ChannelID

			// Validate game action for hit/stand
			game, err := b.validateGameAction(i)
			if err != nil {
				respondWithError(s, i, err.Error())
				return
			}

			switch i.MessageComponentData().CustomID {
			case "hit":
				b.handleGameAction(s, i, game, channelID, func() error {
					return game.Hit(i.Member.User.ID)
				})

			case "stand":
				b.handleGameAction(s, i, game, channelID, func() error {
					return game.Stand(i.Member.User.ID)
				})
			}
		}
	}
}

func (b *Bot) handleGameAction(s *discordgo.Session, i *discordgo.InteractionCreate, game *blackjack.Game, channelID string, action func() error) {
	// Execute the action
	err := action()
	if err != nil && err != blackjack.ErrHandBust {
		respondWithError(s, i, "¡Ay, caramba! Something went wrong: "+err.Error())
		return
	}

	// Check if all players are done
	if game.CheckAllPlayersDone() {
		if !game.CheckAllPlayersBust() {
			// Play dealer's turn - ignore bust since it's part of the game
			game.PlayDealer()
		}
		game.State = entities.StateComplete
	}

	// Update game display with Tuco's dramatic flair
	embed := createGameEmbed(game, s, i.GuildID)
	components := createGameButtons(game)

	// First respond to the interaction
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
	if err != nil {
		// If responding fails, try to edit the message directly
		_, err = s.ChannelMessageEdit(channelID, i.Message.ID, "¡Ay, caramba! Something went wrong with the interaction.")
		if err != nil {
			log.Printf("Error updating message: %v", err)
		}
	}

	// Clean up if game is complete
	if game.State == entities.StateComplete {
		b.mu.Lock()
		delete(b.games, channelID)
		delete(b.lobbies, channelID)
		b.mu.Unlock()
	}
}

func (b *Bot) validateGameAction(i *discordgo.InteractionCreate) (*blackjack.Game, error) {
	channelID := i.ChannelID
	playerID := i.Member.User.ID

	b.mu.RLock()
	game, exists := b.games[channelID]
	b.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("¡Oye! There's no game running in this channel! Start one with /blackjack")
	}

	playerHand, playing := game.Players[playerID]
	if !playing {
		return nil, fmt.Errorf("¡Eh, amigo! You're not in this game! Wait for it to finish")
	}

	// For hit/stand actions, check if player can still play
	if playerHand.Status != blackjack.StatusPlaying {
		switch playerHand.Status {
		case blackjack.StatusBust:
			return nil, fmt.Errorf("¡Ay, Dios mío! You already bust! Wait for the next game")
		case blackjack.StatusStand:
			return nil, fmt.Errorf("¡Tranquilo! You already stood, amigo! Wait for the next game")
		default:
			return nil, fmt.Errorf("¡No más! You can't play right now! Wait for the next game")
		}
	}

	return game, nil
}

func (b *Bot) handleBlackjackCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	channelID := i.ChannelID

	// Check if there's already a game or lobby in this channel
	b.mu.RLock()
	_, gameExists := b.games[channelID]
	_, lobbyExists := b.lobbies[channelID]
	b.mu.RUnlock()

	if gameExists {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "¡Ay caramba! *adjusts golden rings nervously* There's already a game in progress! Wait for it to finish, ¿eh?",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Printf("Error responding to command: %v", err)
		}
		return
	}

	if lobbyExists {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "¡Oye! *polishes cards frantically* There's already a lobby in this channel! Join that one instead!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Printf("Error responding to command: %v", err)
		}
		return
	}

	// Create new lobby
	lobby := &GameLobby{
		OwnerID: i.Member.User.ID,
		Players: make(map[string]bool),
	}
	lobby.Players[i.Member.User.ID] = true

	// Store the lobby
	b.mu.Lock()
	b.lobbies[channelID] = lobby
	b.mu.Unlock()

	// Respond immediately with initial message and lobby state
	embed := createLobbyEmbed(lobby)
	components := createLobbyButtons(lobby.OwnerID)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    "¡Bienvenidos! *Tuco shuffles the cards with flair* Who's ready to play?",
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
	if err != nil {
		log.Printf("Error responding with lobby: %v", err)
		b.mu.Lock()
		delete(b.lobbies, channelID)
		b.mu.Unlock()
	}
}

func (b *Bot) displayGameState(s *discordgo.Session, i *discordgo.InteractionCreate, gameState interface{}) {
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
