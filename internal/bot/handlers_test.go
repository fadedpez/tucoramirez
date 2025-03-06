package bot

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	discordmock "github.com/fadedpez/tucoramirez/internal/discord/mock"
	"github.com/fadedpez/tucoramirez/internal/games"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type HandlersTestSuite struct {
	suite.Suite
	bot        *Bot
	session    *discordmock.SessionHandler
	mockGame   *games.MockGame
	mockManager *games.MockManager
}

func TestHandlersSuite(t *testing.T) {
	suite.Run(t, new(HandlersTestSuite))
}

func (s *HandlersTestSuite) SetupTest() {
	s.session = &discordmock.SessionHandler{}
	s.session.Test(s.T())
	s.mockGame = &games.MockGame{}
	s.mockManager = &games.MockManager{}
	s.bot = &Bot{
		managers: make(map[string]games.Manager),
	}
	s.bot.managers["blackjack"] = s.mockManager
}

func (s *HandlersTestSuite) TestHandleSlashCommand_GameCommand() {
	// Setup
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "blackjack",
			},
		},
	}

	s.mockManager.On("HandleStart", s.session, interaction).Return()

	// Execute
	s.bot.handleSlashCommand(s.session, interaction)

	// Assert
	s.mockManager.AssertExpectations(s.T())
}

func (s *HandlersTestSuite) TestHandleSlashCommand_DuelTuco() {
	// Setup
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "dueltuco",
			},
		},
	}

	s.session.On("InteractionRespond", mock.Anything, mock.Anything).Return(nil)

	// Execute
	s.bot.handleSlashCommand(s.session, interaction)

	// Assert
	s.session.AssertExpectations(s.T())
}

func (s *HandlersTestSuite) TestHandleSlashCommand_Unknown() {
	// Setup
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "unknown_command",
			},
		},
	}

	s.session.On("InteractionRespond", mock.Anything, mock.Anything).Return(nil)

	// Execute
	s.bot.handleSlashCommand(s.session, interaction)

	// Assert
	s.session.AssertExpectations(s.T())
}

func (s *HandlersTestSuite) TestHandleButton_BlackjackGame() {
	// Setup
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionMessageComponent,
			Data: discordgo.MessageComponentInteractionData{
				CustomID: "blackjack_hit",
			},
		},
	}

	s.mockManager.On("HandleButton", s.session, interaction).Return()

	// Execute
	s.bot.handleButton(s.session, interaction)

	// Assert
	s.mockManager.AssertExpectations(s.T())
}

func (s *HandlersTestSuite) TestHandleButton_DuelGame() {
	// Setup
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionMessageComponent,
			Data: discordgo.MessageComponentInteractionData{
				CustomID: "duel_attack",
			},
		},
	}

	s.session.On("InteractionRespond", mock.Anything, mock.Anything).Return(nil)

	// Execute
	s.bot.handleButton(s.session, interaction)

	// Assert
	s.session.AssertExpectations(s.T())
}

func (s *HandlersTestSuite) TestHandleButton_InvalidFormat() {
	// Setup
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionMessageComponent,
			Data: discordgo.MessageComponentInteractionData{
				CustomID: "invalid_button_id",
			},
		},
	}

	s.session.On("InteractionRespond", mock.Anything, mock.Anything).Return(nil)

	// Execute
	s.bot.handleButton(s.session, interaction)

	// Assert
	s.session.AssertExpectations(s.T())
}

func (s *HandlersTestSuite) TestHandleMessage_TucoQuestion() {
	// Setup
	message := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			Content:   "tuco?",
			ChannelID: "test_channel",
		},
	}

	expectedResponse := "¿Qué pasa?"
	s.session.On("ChannelMessageSend", "test_channel", expectedResponse).Return(&discordgo.Message{}, nil)

	// Execute
	s.bot.handleMessage(s.session, message)

	// Assert
	s.session.AssertExpectations(s.T())
}

func (s *HandlersTestSuite) TestHandleMessage_Help() {
	// Setup
	message := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			Content:   "!help",
			ChannelID: "test_channel",
		},
	}

	expectedResponse := "Available commands:\n- tuco?\n- !help"
	s.session.On("ChannelMessageSend", "test_channel", expectedResponse).Return(&discordgo.Message{}, nil)

	// Execute
	s.bot.handleMessage(s.session, message)

	// Assert
	s.session.AssertExpectations(s.T())
}

func (s *HandlersTestSuite) TestHandleMessage_Unknown() {
	// Setup
	message := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			Content:   "unknown_command",
			ChannelID: "test_channel",
		},
	}

	// Execute
	s.bot.handleMessage(s.session, message)

	// Assert - No interactions should occur
	s.session.AssertNotCalled(s.T(), "ChannelMessageSend")
}
