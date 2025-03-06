package games

import (
	"fmt"
	"sync"

	"github.com/fadedpez/tucoramirez/internal/discord"
	"github.com/fadedpez/tucoramirez/internal/types"
)

// Registry manages game factories and their associated commands
type Registry struct {
	factories map[string]Factory
	mu        sync.RWMutex
}

// NewRegistry creates a new game registry
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]Factory),
	}
}

// RegisterGame registers a game factory with the registry
func (r *Registry) RegisterGame(name string, factory Factory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		return types.NewGameError(types.ErrInvalidAction, fmt.Sprintf("Game %s is already registered", name))
	}

	r.factories[name] = factory
	return nil
}

// GetFactory returns the factory for a given game name
func (r *Registry) GetFactory(name string) (Factory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.factories[name]
	if !exists {
		return nil, types.NewGameError(types.ErrGameNotFound, fmt.Sprintf("Game %s not found", name))
	}

	return factory, nil
}

// CreateManager creates a new manager for a given game
func (r *Registry) CreateManager(name string, session discord.SessionHandler) (Manager, error) {
	factory, err := r.GetFactory(name)
	if err != nil {
		return nil, err
	}

	return factory.CreateManager(), nil
}

// ListGames returns a list of registered game names
func (r *Registry) ListGames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	games := make([]string, 0, len(r.factories))
	for name := range r.factories {
		games = append(games, name)
	}
	return games
}
