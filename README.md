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
│   │   ├── wallet.go    # Wallet structure for currency management
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
│   │   ├── wallet/   # Wallet persistence
│   │   │   ├── interface.go  # Repository interface
│   │   │   ├── memory.go     # In-memory implementation
│   │   │   └── sqlite.go     # SQLite implementation
│   │   └── session/ # Game session persistence
│   │       ├── interface.go
│   │       └── memory.go
│   │
│   ├── services/    # Business logic
│   │   ├── game.go  # Generic game operations
│   │   ├── blackjack/  # Blackjack specific
│   │   │   ├── rules.go    # Game rules
│   │   │   └── service.go  # Game operations
│   │   ├── wallet/     # Wallet service
│   │   │   └── service.go  # Wallet operations
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
   - Pure data structures (Game, Card, Hand, Player, Wallet)
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
   - Wallet service for currency management

4. Discord Layer (Presentation)
   - Handles all Discord-specific logic
   - Single command with button interactions
   - Uses service layer for game operations
   - No direct access to repositories

## Wallet System

The bot includes a comprehensive wallet system that allows players to manage their in-game currency.

### Wallet Features

- **Balance Management**: Players can view their current balance
- **Loan System**: Players can take loans to increase their balance
- **Repayment System**: Players can repay their loans
- **Transaction History**: All currency movements are recorded as transactions

### Wallet Entity

```go
type Wallet struct {
    UserID      string
    Balance     int64
    LoanAmount  int64
    LastUpdated time.Time
}

type Transaction struct {
    ID           string
    UserID       string
    Amount       int64
    Type         TransactionType
    Description  string
    Timestamp    time.Time
    BalanceAfter int64
}

type TransactionType string

const (
    TransactionTypeLoan      TransactionType = "loan"
    TransactionTypeRepayment TransactionType = "repayment"
    TransactionTypeBet       TransactionType = "bet"
    TransactionTypeWin       TransactionType = "win"
    TransactionTypeRefund    TransactionType = "refund"
)
```

### Wallet Service

The wallet service provides the following operations:

- **GetOrCreateWallet**: Retrieves a user's wallet or creates one if it doesn't exist
- **AddFunds**: Adds funds to a user's wallet
- **RemoveFunds**: Removes funds from a user's wallet if sufficient funds exist
- **TakeLoan**: Adds a loan amount to the user's wallet
- **RepayLoan**: Repays a portion of the user's loan

### Integration with Games

The wallet system integrates with the blackjack game to:

1. **Place Bets**: When a player starts a game, funds are removed from their wallet
2. **Award Winnings**: When a player wins, funds are added to their wallet
3. **Refund Bets**: In certain game scenarios, bets may be refunded to the wallet

Players can use the `/wallet` command to view their balance and manage loans.

## Gameplay Features

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
