# Tuco Ramirez Discord Bot

A Discord bot for playing card games, featuring the charismatic personality of Tuco Ramirez from "The Good, the Bad and the Ugly".

## Project Structure

```
tucoramirez/
├── .env               # Discord token and config
├── cmd/
│   ├── bot/          # Main bot entry point
│   │   └── main.go   # Bot startup, env loading
│   └── migration/    # Database migration tool
│       └── main.go   # Migration helper script
│
├── migrations/       # SQLite migration files
│   └── 001_initial_schema.sql  # Initial database schema
│
├── pkg/
│   ├── db/           # Database utilities
│   │   └── migrations/  # Migration system
│   │       └── migrations.go  # Migration framework
│   │
│   ├── entities/     # Pure data structures
│   │   ├── types.go     # Shared types (ID types, enums)
│   │   ├── errors.go    # Entity-specific errors
│   │   ├── card.go      # Card, Deck structures
│   │   ├── game.go      # Game structures
│   │   ├── image.go     # Image structure for game images
│   │   └── player.go    # Player structures
│   │
│   ├── repositories/ # Data persistence
│   │   ├── game/     # Game persistence
│   │   │   ├── interface.go  # Repository interface
│   │   │   ├── memory.go     # In-memory implementation
│   │   │   └── sqlite.go     # SQLite implementation
│   │   ├── player/   # Player persistence
│   │   │   ├── interface.go
│   │   │   └── memory.go
│   │   └── session/ # Game session persistence
│   │       ├── interface.go
│   │       └── memory.go
│   │
│   ├── services/    # Business logic
│   │   ├── game.go  # Generic game operations
│   │   ├── blackjack/  # Blackjack specific
│   │   │   ├── rules.go    # Game rules
│   │   │   └── service.go  # Game operations
│   │   └── image/     # Image service
│   │       └── service.go  # Image operations
│   │
│   └── discord/     # Discord interface
│       ├── bot.go   # Bot setup and configuration
│       ├── config/  # Discord-specific configuration
│       │   └── config.go  # Env loading, bot config
│       ├── client/  # Discord client wrapper
│       │   └── client.go  # Discord session management
│       ├── commands/  # Command handlers (future)
│       └── handlers/  # Event handlers (future)
│
└── go.mod
```

## Architecture

1. Entities Layer (Core)
   - Pure data structures (Game, Card, Hand, Player)
   - No business logic
   - Used across all layers
   - Defines core types and states

2. Repository Layer
   - Implements data persistence and retrieval
   - Thread-safe in-memory storage
   - SQLite implementation for persistent storage
   - Database migration system for schema evolution
   - One repository per entity type
   - Clean interfaces for data access

3. Service Layer
   - Uses repositories to manage game state
   - Implements game operations (hit, stand)
   - Coordinates between repositories and presentation
   - Contains blackjack rules and logic
   - Image service for game completion images

4. Discord Layer (Presentation)
   - Handles all Discord-specific logic
   - Single command with button interactions
   - Uses service layer for game operations
   - No direct access to repositories

### Entities Layer
- Pure data structures
- No business logic
- Used across all layers

#### Core Entities

1. Game Entity
```go
type Game struct {
    ID          GameID
    Type        GameType        // "blackjack", future: "poker", etc
    State       GameState       // betting, dealing, playerTurn, etc
    PlayerIDs   []PlayerID      // References to players
    DealerID    PlayerID        // Reference to dealer
    Deck        Deck
    Round       int             // Current round number
    ActiveID    PlayerID        // Current player's turn
    Bets        map[PlayerID]int
}

type GameState string

const (
    GameStateBetting    GameState = "betting"
    GameStateDealing    GameState = "dealing"
    GameStatePlayerTurn GameState = "playerTurn"
    GameStateDealerTurn GameState = "dealerTurn"
    GameStateComplete   GameState = "complete"
)
```

2. Card Entity
```go
type Card struct {
    Suit  Suit
    Rank  Rank
    Value int
}

type Deck struct {
    Cards []Card
}
```

3. Player Entity
```go
type Player struct {
    ID      PlayerID
    Name    string
    Balance int
}

type Hand struct {
    Cards []Card
    Bet   int
}
```

### Repository Layer

#### Repository Interfaces

```go
type Repository interface {
    // Core operations
    Create(game *entities.Game) error
    Get(id entities.GameID) (*entities.Game, error)
    Update(game *entities.Game) error
    Delete(id entities.GameID) error
    
    // Game results
    SaveGameResult(ctx context.Context, result *entities.GameResult) error
    GetPlayerResults(ctx context.Context, playerID string) ([]*entities.GameResult, error)
    GetChannelResults(ctx context.Context, channelID string, limit int) ([]*entities.GameResult, error)
}
```

#### SQLite Implementation

The SQLite repository provides persistent storage for game data, including:

- Deck state persistence
- Game results tracking
- Player results history

```go
func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
    // Ensure directory exists
    dbDir := filepath.Dir(dbPath)
    if err := os.MkdirAll(dbDir, 0755); err != nil {
        return nil, fmt.Errorf("error creating database directory: %w", err)
    }

    // Open database
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, fmt.Errorf("error opening database: %w", err)
    }

    // Apply migrations
    migrator := migrations.NewMigrator(db, "migrations")
    if err := migrator.MigrateUp(); err != nil {
        db.Close()
        return nil, fmt.Errorf("error applying migrations: %w", err)
    }

    return &SQLiteRepository{db: db}, nil
}
```

#### Database Migration System

The bot includes a database migration system to manage schema changes:

1. **Migration Files**: SQL files in the `migrations/` directory that define schema changes
2. **Automatic Migration**: Applied when the bot starts
3. **Migration Helper**: Command-line tool for creating and applying migrations

**Creating a New Migration**:

```bash
# Create a new migration file
go run cmd/migration/main.go create "add wallet tables"
```

This creates a numbered migration file (e.g., `002_add_wallet_tables.sql`) with SQL templates and examples.

**Applying Migrations**:

Migrations are automatically applied when the bot starts, but can also be applied manually:

```bash
# Apply pending migrations
go run cmd/migration/main.go migrate
```

### Service Layer

#### Game Service

```go
func (s *Service) Hit(gameID GameID, playerID PlayerID) (*GameResult, error) {
    // Get current game state
    game, err := s.games.Get(gameID)
    if err != nil {
        return nil, err
    }
    
    // Check if it's the player's turn
    if game.ActiveID != playerID {
        return nil, ErrNotPlayerTurn
    }
    
    // Deal a card to the player
    card, err := game.Deck.Draw()
    if err != nil {
        return nil, err
    }
    
    // Add card to player's hand
    player := game.GetPlayer(playerID)
    player.Hand.AddCard(card)
    
    // Check if player busts
    if player.Hand.Value() > 21 {
        // End player's turn
        game.NextPlayer()
    }
    
    // Update game state
    if err := s.games.Update(game); err != nil {
        return nil, err
    }
    
    return &GameResult{
        Game:   game,
        Status: "hit",
    }, nil
}
```

#### Image Service

The image service provides random images to display when a game completes:

```go
// GetRandomImage returns a random image from the collection
func (s *Service) GetRandomImage() *entities.Image {
    if len(s.images) == 0 {
        return &entities.Image{URL: ""} // Return empty image if none available
    }
    
    randomIndex := s.rng.Intn(len(s.images))
    return s.images[randomIndex]
}
```

### Game State Architecture

The project uses a flexible architecture for handling game states and results across different game types:

#### Generic Game States

Defined in `entities/gamestate.go`, these are game-agnostic states and results:

```go
// Game state types
type GameState string

const (
    StateWaiting  GameState = "WAITING"
    StateDealing  GameState = "DEALING"
    StatePlaying  GameState = "PLAYING"
    StateDealer   GameState = "DEALER"
    StateComplete GameState = "COMPLETE"
)

// Base result types
type Result string

const (
    ResultWin  Result = "WIN"
    ResultLose Result = "LOSE"
    ResultPush Result = "PUSH"
)
```

#### GameDetails Interface

This interface allows for game-specific details while maintaining a consistent framework:

```go
// GameDetails defines what game-specific result details must provide
type GameDetails interface {
    // GameType returns the type of game (ex blackjack or poker)
    GameType() GameState
    // ValidateDetails ensures the details are valid for the game
    ValidateDetails() error
}
```

#### Game-Specific Implementations

Each game implements its own version of GameDetails:

```go
// BlackjackDetails contains game-specific result details
type BlackjackDetails struct {
    DealerScore int
    IsBlackjack bool
    IsBust      bool
}

func (d *BlackjackDetails) GameType() entities.GameState {
    return entities.StateDealing // will be updated to a game type constant
}

func (d *BlackjackDetails) ValidateDetails() error {
    if d.DealerScore < 0 || d.DealerScore > 31 {
        return errors.New("invalid dealer score")
    }
    return nil
}
```

This architecture allows us to:
1. Handle multiple card games with a consistent state framework
2. Capture game-specific details and rules
3. Process results uniformly at the repository level
4. Add new games without modifying core game state logic

### Gameplay Features

### Blackjack

#### Turn-Based Player Actions
The blackjack game implements a turn-based system for player actions:

- Players take turns in the order they joined the game
- A pointing finger emoji (👉) indicates whose turn it is currently
- Only the current player can take actions (hit/stand)
- Players who try to act out of turn receive a friendly message
- The turn automatically advances after a player stands or busts
- Once all players have completed their turns, the dealer plays according to house rules
- The `PlayerOrder` slice maintains consistent player ordering for both turns and UI display

This system ensures fair gameplay while maintaining the social dynamics of a card game. The visual indicator makes it clear whose turn it is without cluttering the interface.

```go
// Example of turn management in the Game struct
type Game struct {
    // ... other fields
    PlayerOrder []string // Ordered list of player IDs
    CurrentTurn int      // Index into PlayerOrder
}

// IsPlayerTurn checks if it's the specified player's turn
func (g *Game) IsPlayerTurn(playerID string) bool {
    currentPlayer, err := g.GetCurrentTurnPlayerID()
    if err != nil {
        return false
    }
    return playerID == currentPlayer
}

// In UI creation, use PlayerOrder for consistent display
// Instead of: for playerID, hand := range game.Players { ... }
// Use: for _, playerID := range game.PlayerOrder { hand := game.Players[playerID]; ... }
```

### Discord Layer

```go
func NewBot(token string, repository game.Repository) (*Bot, error) {
    // Create Discord session
    session, err := discordgo.New("Bot " + token)
    if err != nil {
        return nil, fmt.Errorf("failed to create Discord session: %w", err)
    }
    
    // Initialize image service
    imageService, err := image.NewService("images.txt")
    if err != nil {
        return nil, fmt.Errorf("error initializing image service: %w", err)
    }
    
    bot := &Bot{
        session:      session,
        token:        token,
        games:        make(map[string]*blackjack.Game),
        lobbies:      make(map[string]*GameLobby),
        repo:         repository,
        imageService: imageService,
    }
    
    // Register handlers
    bot.registerHandlers()
    
    return bot, nil
}
```

## Implementation Order

The project follows this implementation order:

1. **Repository Layer** (Current)
   - SQLite implementation for persistent storage
   - Migration system for schema evolution

2. **Wallet System** (Next)
   - Currency tracking per player
   - Add/remove funds operations

3. **Loan System**
   - Track loans as positive integers
   - Display as negative balances

4. **Betting System**
   - Initial ante betting
   - Win/loss payouts
   - Special bets (double down, split, insurance)

5. **Advanced Game Features**
   - Split hands
   - Insurance bets
   - Multiple concurrent games

## Development

### Setup

1. Clone the repository
2. Create a `.env` file with your Discord bot token:
   ```
   DISCORD_TOKEN=your_token_here
   ```
3. Run the bot:
   ```bash
   go run cmd/bot/main.go
   ```

### Database Migrations

When adding new features that require database changes:

1. Create a new migration:
   ```bash
   go run cmd/migration/main.go create "description of changes"
   ```

2. Edit the generated SQL file in the `migrations/` directory

3. The changes will be automatically applied when the bot starts
