package types

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ErrorTestSuite struct {
	suite.Suite
}

func TestErrorSuite(t *testing.T) {
	suite.Run(t, new(ErrorTestSuite))
}

func (s *ErrorTestSuite) TestNewGameError() {
	// Setup
	code := ErrGameNotFound
	message := "game not found"

	// Execute
	err := NewGameError(code, message)

	// Assert
	s.Equal(code, err.Code, "Error code should match")
	s.Equal(message, err.Message, "Error message should match")
	s.Nil(err.Err, "Underlying error should be nil")
}

func (s *ErrorTestSuite) TestWrapError() {
	// Setup
	code := ErrInternalError
	message := "database error"
	underlying := errors.New("connection failed")

	// Execute
	err := WrapError(code, message, underlying)

	// Assert
	s.Equal(code, err.Code, "Error code should match")
	s.Equal(message, err.Message, "Error message should match")
	s.Equal(underlying, err.Err, "Underlying error should match")
}

func (s *ErrorTestSuite) TestErrorString() {
	testCases := []struct {
		name     string
		err      *GameError
		expected string
	}{
		{
			name:     "Simple error",
			err:      NewGameError(ErrGameNotFound, "game not found"),
			expected: "GAME_NOT_FOUND: game not found",
		},
		{
			name:     "Wrapped error",
			err:      WrapError(ErrInternalError, "database error", errors.New("connection failed")),
			expected: "INTERNAL_ERROR: database error (connection failed)",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.Equal(tc.expected, tc.err.Error(), "Error string should match expected format")
		})
	}
}

func (s *ErrorTestSuite) TestIsGameError() {
	// Setup
	gameErr := NewGameError(ErrGameNotFound, "game not found")
	regularErr := errors.New("regular error")

	// Test cases
	testCases := []struct {
		name     string
		err      error
		code     ErrorCode
		expected bool
	}{
		{
			name:     "Matching game error",
			err:      gameErr,
			code:     ErrGameNotFound,
			expected: true,
		},
		{
			name:     "Non-matching game error",
			err:      gameErr,
			code:     ErrInternalError,
			expected: false,
		},
		{
			name:     "Regular error",
			err:      regularErr,
			code:     ErrGameNotFound,
			expected: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			code:     ErrGameNotFound,
			expected: false,
		},
	}

	// Execute and assert
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := IsGameError(tc.err, tc.code)
			s.Equal(tc.expected, result, "IsGameError result should match expected value")
		})
	}
}

func (s *ErrorTestSuite) TestAs() {
	// Setup
	gameErr := NewGameError(ErrGameNotFound, "game not found")
	regularErr := errors.New("regular error")

	// Test cases
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Game error",
			err:      gameErr,
			expected: true,
		},
		{
			name:     "Regular error",
			err:      regularErr,
			expected: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
	}

	// Execute and assert
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			var target *GameError
			result := As(tc.err, &target)
			s.Equal(tc.expected, result, "As result should match expected value")
			if tc.expected {
				s.Equal(gameErr, target, "Target should be set to the game error")
			}
		})
	}
}
