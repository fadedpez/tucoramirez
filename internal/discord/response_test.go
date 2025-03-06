package discord

import (
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"
	discordmock "github.com/fadedpez/tucoramirez/internal/discord/mock"
	"github.com/fadedpez/tucoramirez/internal/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ResponseTestSuite struct {
	suite.Suite
	session *discordmock.SessionHandler
}

func TestResponseSuite(t *testing.T) {
	suite.Run(t, new(ResponseTestSuite))
}

func (s *ResponseTestSuite) SetupTest() {
	s.session = &discordmock.SessionHandler{}
	s.session.Test(s.T())
}

func (s *ResponseTestSuite) TestNewResponse() {
	// Setup
	content := "test content"
	components := []discordgo.MessageComponent{
		discordgo.Button{
			Label: "Test Button",
			Style: discordgo.PrimaryButton,
		},
	}

	// Execute
	resp := NewResponse(content, components)

	// Assert
	s.NotNil(resp)
	s.Equal(content, resp.Content)
	s.Equal(components, resp.Components)
	s.False(resp.Ephemeral)
}

func (s *ResponseTestSuite) TestNewEphemeralResponse() {
	// Setup
	content := "test content"
	components := []discordgo.MessageComponent{
		discordgo.Button{
			Label: "Test Button",
			Style: discordgo.PrimaryButton,
		},
	}

	// Execute
	resp := NewEphemeralResponse(content, components)

	// Assert
	s.NotNil(resp)
	s.Equal(content, resp.Content)
	s.Equal(components, resp.Components)
	s.True(resp.Ephemeral)
}

func (s *ResponseTestSuite) TestNewErrorResponse() {
	testCases := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "❌ An error occurred: <nil>",
		},
		{
			name:     "simple error",
			err:      errors.New("test error"),
			expected: "❌ An error occurred: test error",
		},
		{
			name:     "game error",
			err:      types.NewGameError(types.ErrInvalidState, "game error"),
			expected: "⚠️ game error",
		},
		{
			name:     "wrapped error",
			err:      errors.New("wrapped: test error"),
			expected: "❌ An error occurred: wrapped: test error",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Execute
			resp := NewErrorResponse(tc.err)

			// Assert
			s.NotNil(resp)
			s.Equal(tc.expected, resp.Content)
			s.True(resp.Ephemeral)
		})
	}
}

func (s *ResponseTestSuite) TestSendResponse() {
	// Setup
	content := "test content"
	components := []discordgo.MessageComponent{
		discordgo.Button{
			Label: "Test Button",
			Style: discordgo.PrimaryButton,
		},
	}
	resp := NewResponse(content, components)

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:   "test_interaction",
			Type: discordgo.InteractionApplicationCommand,
		},
	}

	s.session.On("InteractionRespond", interaction.Interaction, mock.Anything).Return(nil)

	// Execute
	err := SendResponse(s.session, interaction, resp)

	// Assert
	s.NoError(err)
	s.session.AssertExpectations(s.T())
}

func (s *ResponseTestSuite) TestUpdateResponse() {
	// Setup
	content := "updated content"
	components := []discordgo.MessageComponent{
		discordgo.Button{
			Label: "Test Button",
			Style: discordgo.PrimaryButton,
		},
	}
	resp := NewResponse(content, components)

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:   "test_interaction",
			Type: discordgo.InteractionApplicationCommand,
		},
	}

	s.session.On("InteractionRespond", interaction.Interaction, mock.Anything).Return(nil)

	// Execute
	err := UpdateResponse(s.session, interaction, resp)

	// Assert
	s.NoError(err)
	s.session.AssertExpectations(s.T())
}

func (s *ResponseTestSuite) TestSendGameResponse() {
	// Setup
	content := "game content"
	components := []discordgo.MessageComponent{
		discordgo.Button{
			Label: "Test Button",
			Style: discordgo.PrimaryButton,
		},
	}

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:   "test_interaction",
			Type: discordgo.InteractionApplicationCommand,
		},
	}

	s.session.On("InteractionRespond", interaction.Interaction, mock.Anything).Return(nil)

	// Execute
	err := SendGameResponse(s.session, interaction, content, components)

	// Assert
	s.NoError(err)
	s.session.AssertExpectations(s.T())
}

func (s *ResponseTestSuite) TestSendErrorResponse() {
	// Setup
	testErr := errors.New("test error")
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:   "test_interaction",
			Type: discordgo.InteractionApplicationCommand,
		},
	}

	s.session.On("InteractionRespond", interaction.Interaction, mock.Anything).Return(nil)

	// Execute
	err := SendErrorResponse(s.session, interaction, testErr)

	// Assert
	s.NoError(err)
	s.session.AssertExpectations(s.T())
}
