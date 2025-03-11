package discord

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/pkg/services/blackjack"
)

func (b *Bot) handleReady(s *discordgo.Session, r *discordgo.Ready) {
	// Register only the blackjack command
	command := &discordgo.ApplicationCommand{
		Name:        "blackjack",
		Description: "¡Juega blackjack conmigo, amigo! Start a new game of blackjack",
	}

	_, err := s.ApplicationCommandCreate(s.State.User.ID, "", command)
	if err != nil {
		log.Printf("Error creating command %v: %v", command.Name, err)
	}

	// Add component interaction handler for buttons
	s.AddHandler(b.handleInteractions)
}

func (b *Bot) handleInteractions(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		if i.ApplicationCommandData().Name == "blackjack" {
			handleBlackjackCommand(b, s, i)
		}
	case discordgo.InteractionMessageComponent:
		handleButtonInteraction(b, s, i)
	}
}

func handleBlackjackCommand(b *Bot, s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Start new game
	game := blackjack.NewGame(i.ChannelID)
	b.games[i.ChannelID] = game

	// Add the player
	if err := game.AddPlayer(i.Member.User.ID); err != nil {
		respondWithError(s, i, err.Error())
		return
	}

	// Start the game
	if err := game.Start(); err != nil {
		respondWithError(s, i, err.Error())
		return
	}

	// Show initial game state with buttons
	b.displayGameState(s, i, game)
}

func handleButtonInteraction(b *Bot, s *discordgo.Session, i *discordgo.InteractionCreate) {
	game, exists := b.games[i.ChannelID]
	if !exists {
		respondWithError(s, i, "¡No hay juego activo! Start a new game with /blackjack")
		return
	}

	switch i.MessageComponentData().CustomID {
	case "hit":
		if err := game.Hit(i.Member.User.ID); err != nil {
			respondWithError(s, i, err.Error())
			return
		}
		b.displayGameState(s, i, game)

	case "stand":
		if err := game.Stand(i.Member.User.ID); err != nil {
			respondWithError(s, i, err.Error())
			return
		}

		// Play dealer's turn
		if err := game.PlayDealer(); err != nil {
			respondWithError(s, i, err.Error())
			return
		}

		b.displayGameState(s, i, game)
	}
}

// validateGameAction checks if a player can perform an action in the current game
func (b *Bot) validateGameAction(s *discordgo.Session, i *discordgo.InteractionCreate) (*blackjack.Game, error) {
	channelID := i.ChannelID
	playerID := i.Member.User.ID

	game, exists := b.games[channelID]
	if !exists {
		return nil, fmt.Errorf("¡Oye! There's no game running in this channel! Start one with /blackjack")
	}

	playerHand, playing := game.Players[playerID]
	if !playing {
		return nil, fmt.Errorf("¡Eh, amigo! You're not in this game! Wait for it to finish")
	}

	if playerHand.Status != blackjack.StatusPlaying {
		switch playerHand.Status {
		case blackjack.StatusBust:
			return nil, fmt.Errorf("¡Ay, Dios mío! You already bust! Wait for the next game")
		case blackjack.StatusStand:
			return nil, fmt.Errorf("¡Tranquilo! You already stood! Wait for the next game")
		}
	}

	return game, nil
}

// Then update handleBlackjackCommand:
func (b *Bot) handleBlackjackCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	channelID := i.ChannelID

	// Check if there's already a game running
	if game, exists := b.games[channelID]; exists {
		if _, playing := game.Players[i.Member.User.ID]; playing {
			respondWithError(s, i, "¡Paciencia! You're already in a game!")
			return
		}
		respondWithError(s, i, "¡Espera! There's already a game in progress!")
		return
	}

	// Create new game
	game := blackjack.NewGame(channelID)

	// Add player to the game
	err := game.AddPlayer(i.Member.User.ID)
	if err != nil {
		respondWithError(s, i, "¡Ay, caramba! Failed to add you to the game: "+err.Error())
		return
	}

	// Deal initial cards
	err = game.Start()
	if err != nil {
		respondWithError(s, i, "¡Madre de Dios! Failed to deal cards: "+err.Error())
		return
	}

	// Store game in bot's game map
	b.games[channelID] = game

	// Display initial game state
	b.displayGameState(s, i, game)
}

// And update handleButton for hit/stand:
func (b *Bot) handleButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Handle play again button
	if i.MessageComponentData().CustomID == "play_again" {
		// Start a new game directly
		b.handleBlackjackCommand(s, i)
		return
	}

	// Validate game action for hit/stand
	game, err := b.validateGameAction(s, i)
	if err != nil {
		respondWithError(s, i, err.Error())
		return
	}

	switch i.MessageComponentData().CustomID {
	case "hit":
		err = game.Hit(i.Member.User.ID)
	case "stand":
		err = game.Stand(i.Member.User.ID)

		// Check if all players are done and at least one player hasn't bust
		if err == nil && game.CheckAllPlayersDone() {
			if !game.CheckAllPlayersBust() {
				// Only play dealer's turn if someone hasn't bust
				err = game.PlayDealer()
			} else {
				// Everyone bust, just mark game as complete
				game.State = blackjack.StateComplete
			}
		}
	}

	if err != nil {
		respondWithError(s, i, "¡Ay, caramba! Something went wrong: "+err.Error())
		return
	}

	// Check if game is over (all players done or dealer finished)
	if game.CheckAllPlayersDone() || game.State == blackjack.StateComplete {
		// Show results instead of game state
		b.displayGameResults(s, i, game)
		// Clean up the game
		delete(b.games, i.ChannelID)
	} else {
		// Show normal game state
		b.displayGameState(s, i, game)
	}
}
