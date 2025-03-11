# Tuco Ramirez Discord Bot

A Discord bot for playing card games, featuring the charismatic personality of Tuco Ramirez from "The Good, the Bad and the Ugly".

## Project Structure

```
tucoramirez/
├── .env               # Discord token and config
├── cmd/
│   └── bot/          # Main bot entry point
│       └── main.go   # Bot startup, env loading
│
├── pkg/
│   ├── entities/     # Pure data structures
│   │   ├── types.go     # Shared types (ID types, enums)
│   │   ├── errors.go    # Entity-specific errors
│   │   ├── card.go      # Card, Deck structures
│   │   ├── game.go      # Game structures
│   │   └── player.go    # Player structures
│   │
│   ├── repositories/ # Data persistence
│   │   ├── game/     # Game persistence
│   │   │   ├── interface.go  # Repository interface
│   │   │   └── memory.go     # In-memory implementation
│   │   ├── player/   # Player persistence
│   │   │   ├── interface.go
│   │   │   └── memory.go
│   │   └── session/ # Game session persistence
│   │       ├── interface.go
│   │       └── memory.go
│   │
│   ├── services/    # Business logic
│   │   ├── game.go  # Generic game operations
│   │   └── blackjack/  # Blackjack specific
│   │       ├── rules.go    # Game rules
│   │       └── service.go  # Game operations
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
   - One repository per entity type
   - Clean interfaces for data access

3. Service Layer
   - Uses repositories to manage game state
   - Implements game operations (hit, stand)
   - Coordinates between repositories and presentation
   - Contains blackjack rules and logic

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
    GameStateResolving  GameState = "resolving"
)
```

2. Player Entity
```go
type Player struct {
    ID       PlayerID
    Name     string
    Hand     Hand
    Balance  int
    Status   PlayerStatus
    IsDealer bool        // Identifies if this is a dealer
}

type PlayerStatus string

const (
    PlayerStatusActive PlayerStatus = "active"
    PlayerStatusBust   PlayerStatus = "bust"
    PlayerStatusStand  PlayerStatus = "stand"
)
```

3. Card Entities
```go
type Card struct {
    Suit Suit
    Rank Rank
}

type Hand struct {
    Cards   []Card
    IsSplit bool    // For blackjack split hands
}

type Deck struct {
    Cards    []Card
    NumDecks int    // For multiple deck games
}

// Card enums
type Suit string
type Rank int

const (
    Hearts   Suit = "hearts"
    Diamonds Suit = "diamonds"
    Clubs    Suit = "clubs"
    Spades   Suit = "spades"
)

const (
    Ace   Rank = 1
    Two   Rank = 2
    // ... other ranks
    King  Rank = 13
)
```

### Repositories Layer
Each entity has its own repository interface and in-memory implementation.

#### Game Repository
```go
// pkg/repositories/game/interface.go
type Repository interface {
    // Core operations
    Create(game *entities.Game) error
    Get(id entities.GameID) (*entities.Game, error)
    Update(game *entities.Game) error
    Delete(id entities.GameID) error
    
    // Query operations
    FindByState(state entities.GameState) ([]*entities.Game, error)
    FindByPlayer(playerID entities.PlayerID) ([]*entities.Game, error)
}

// Memory implementation
type MemoryRepository struct {
    games map[entities.GameID]*entities.Game
    mu    sync.RWMutex
}
```

#### Player Repository
```go
// pkg/repositories/player/interface.go
type Repository interface {
    // Core operations
    Create(player *entities.Player) error
    Get(id entities.PlayerID) (*entities.Player, error)
    Update(player *entities.Player) error
    Delete(id entities.PlayerID) error
    
    // Query operations
    FindByGame(gameID entities.GameID) ([]*entities.Player, error)
    FindByStatus(status entities.PlayerStatus) ([]*entities.Player, error)
}
```

#### Repository Usage in Services
```go
// pkg/services/blackjack/service.go
type Service struct {
    games   repositories.GameRepository
    players repositories.PlayerRepository
}

func (s *Service) CreateGame(hostID entities.PlayerID) (*entities.Game, error) {
    game := entities.NewGame(hostID)
    if err := s.games.Create(game); err != nil {
        return nil, err
    }
    return game, nil
}

func (s *Service) JoinGame(gameID entities.GameID, playerID entities.PlayerID) error {
    game, err := s.games.Get(gameID)
    if err != nil {
        return err
    }
    
    player, err := s.players.Get(playerID)
    if err != nil {
        return err
    }
    
    // Add player to game
    game.PlayerIDs = append(game.PlayerIDs, playerID)
    return s.games.Update(game)
}
```

#### Memory Implementation Details
1. Thread-Safe Operations
```go
func (r *MemoryRepository) Get(id entities.GameID) (*entities.Game, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    if game, exists := r.games[id]; exists {
        return game, nil
    }
    return nil, ErrGameNotFound
}
```

2. Data Consistency
```go
func (r *MemoryRepository) Update(game *entities.Game) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if _, exists := r.games[game.ID]; !exists {
        return ErrGameNotFound
    }
    
    // Create deep copy to prevent external modifications
    r.games[game.ID] = game.Clone()
    return nil
}
```

3. Query Implementation
```go
func (r *MemoryRepository) FindByState(state entities.GameState) ([]*entities.Game, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    var games []*entities.Game
    for _, game := range r.games {
        if game.State == state {
            games = append(games, game.Clone())
        }
    }
    return games, nil
}
```

#### Key Repository Principles
1. Thread Safety
   - All operations are mutex-protected
   - Prevents race conditions in concurrent access

2. Data Isolation
   - Deep copies prevent external mutations
   - Each repository manages its own data

3. Consistent Interface
   - Same CRUD operations across repositories
   - Similar query patterns for all entities

4. Error Handling
   - Well-defined error types
   - Consistent error patterns across repositories

### Services Layer
- Contains all business logic
- Uses entities as data holders
- Coordinates between repositories

#### Blackjack Rules
Located in `pkg/services/blackjack/rules.go`:

1. Game Rules
   - IsBlackjack(hand *entities.Hand) -> bool
   - IsBust(hand *entities.Hand) -> bool
   - CanSplit(hand *entities.Hand) -> bool
   - CanDoubleDown(hand *entities.Hand) -> bool
   - CompareHands(player, dealer *entities.Hand) -> Winner

2. Scoring
   - CalculateScore(hand *entities.Hand) -> int
   - GetPossibleScores(hand *entities.Hand) -> []int
   - GetBestScore(hand *entities.Hand) -> int

3. Actions
   - CanHit(hand *entities.Hand) -> bool
   - CanStand(hand *entities.Hand) -> bool
   - ShouldDealerHit(hand *entities.Hand) -> bool

### Service Initialization

```go
// pkg/services/blackjack/service.go
type Service struct {
    games   repositories.GameRepository
    players repositories.PlayerRepository
    rules   *rules.BlackjackRules
}

// NewService creates a new blackjack service with dependencies
func NewService(games repositories.GameRepository, players repositories.PlayerRepository) *Service {
    return &Service{
        games:   games,
        players: players,
        rules:   rules.New(),
    }
}

// pkg/discord/bot.go
type Bot struct {
    session  *discordgo.Session
    blackjack *blackjack.Service
}

// NewBot creates a new Discord bot with all services
func NewBot(token string) (*Bot, error) {
    // Create Discord session
    session, err := discordgo.New("Bot " + token)
    if err != nil {
        return nil, fmt.Errorf("failed to create Discord session: %w", err)
    }

    // Initialize repositories
    games := memory.NewGameRepository()
    players := memory.NewPlayerRepository()

    // Initialize services
    blackjackService := blackjack.NewService(games, players)

    return &Bot{
        session:   session,
        blackjack: blackjackService,
    }, nil
}

// pkg/main.go
func main() {
    // Load environment variables
    token := os.Getenv("DISCORD_TOKEN")
    if token == "" {
        log.Fatal("DISCORD_TOKEN environment variable is required")
    }

    // Create and start bot
    bot, err := discord.NewBot(token)
    if err != nil {
        log.Fatalf("Failed to create bot: %v", err)
    }

    // Start bot (blocking call)
    if err := bot.Start(); err != nil {
        log.Fatalf("Bot error: %v", err)
    }
}

### Discord Interaction Design

### Command Structure
Single entry point via `/blackjack` command that manages the entire game flow through button interactions.

### Game Message Flow
The bot maintains a single, updating message throughout the game:

```
/blackjack
└── Game Message (Single message that updates)
    ├── Lobby Phase
    │   ├── Title: "Tuco's Blackjack Table"
    │   ├── Description: "Who's brave enough to join? (0/8 players)"
    │   ├── Fields: 
    │   │   └── Players: List of joined players
    │   └── Buttons:
    │       ├── Join 🃏
    │       └── Start 🎲 (Host only)
    │
    ├── Playing Phase
    │   ├── Fields:
    │   │   ├── Dealer: "Shows: 🂮 [Hidden]"
    │   │   └── Players: Each player's cards and status
    │   │       Example: "Frank: 🂮 🂫 (15) - Playing"
    │   │       Example: "Alice: 🂭 🂪 (20) - Standing"
    │   └── Buttons:
    │       ├── Hit 👆
    │       └── Stand ✋
    │
    └── Game Over
        ├── Description: "Game Over!"
        ├── Fields:
        │   ├── Dealer: "Cards: 🂮 🂫 (18)"
        │   └── Results: Shows all hands and who won/lost
        └── Button:
            └── Play Again 🔄 (Anyone can click)
```

### Message Components

#### Button IDs
```go
// pkg/discord/components/ids.go
const (
    // Lobby Phase
    ButtonJoinGame  = "btn_join"    // Join the game
    ButtonStartGame = "btn_start"   // Start the game (host only)
    
    // Playing Phase
    ButtonHit      = "btn_hit"      // Hit for another card
    ButtonStand    = "btn_stand"    // Stand with current hand
    
    // Game Over Phase
    ButtonPlayAgain = "btn_again"   // Start new game
)
```

#### Component Data Structure
```go
// Each button includes game context in its custom ID
type ButtonID struct {
    Action  string    // From constants above
    GameID  string    // Current game ID
    Data    string    // Optional extra data
}

// Example encoded ID: "btn_hit:game123:"
func EncodeButtonID(action, gameID, data string) string {
    return fmt.Sprintf("%s:%s:%s", action, gameID, data)
}

func DecodeButtonID(customID string) ButtonID {
    parts := strings.Split(customID, ":")
    return ButtonID{
        Action:  parts[0],
        GameID:  parts[1],
        Data:    parts[2],
    }
}
```

#### Message Builders
```go
// pkg/discord/messages/builders.go
func BuildLobbyMessage(game *entities.Game, players []*entities.Player) *discordgo.MessageEmbed {
    // Create base embed
    embed := &discordgo.MessageEmbed{
        Title:       "Tuco's Blackjack Table",
        Description: fmt.Sprintf("Who's brave enough to join? (%d/8 players)", len(players)),
        Fields:      buildPlayerList(players),
    }

    // Add action row with buttons
    buttons := []discordgo.MessageComponent{
        discordgo.Button{
            CustomID: EncodeButtonID(ButtonJoinGame, game.ID, ""),
            Label:    "Join 🃏",
            Style:    discordgo.PrimaryButton,
        },
        discordgo.Button{
            CustomID: EncodeButtonID(ButtonStartGame, game.ID, ""),
            Label:    "Start 🎲",
            Style:    discordgo.SuccessButton,
        },
    }

    return embed
}

func BuildGameMessage(game *entities.Game, players []*entities.Player) *discordgo.MessageEmbed {
    // Similar pattern for game state...
}
```

#### Handler Registration
```go
// pkg/discord/handlers/buttons.go
func (b *Bot) registerButtonHandlers() {
    b.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
        if i.Type != discordgo.InteractionMessageComponent {
            return
        }

        // Decode button ID
        btn := DecodeButtonID(i.MessageComponentData().CustomID)
        
        // Route to appropriate handler
        switch btn.Action {
        case ButtonJoinGame:
            b.handleJoinGame(s, i, btn)
        case ButtonStartGame:
            b.handleStartGame(s, i, btn)
        case ButtonHit:
            b.handleHit(s, i, btn)
        case ButtonStand:
            b.handleStand(s, i, btn)
        case ButtonPlayAgain:
            b.handlePlayAgain(s, i, btn)
        }
    })
}
```

Key Design Points:
1. Consistent Button IDs
   - Clear naming convention
   - Includes game context
   - Easy to extend

2. Message Building
   - Centralized message creation
   - Consistent styling
   - Reusable components

3. Handler Organization
   - Clean routing based on action
   - Access to game context
   - Error handling at each level

### Message Formatting

#### Game Messages
```
# Lobby Message
🎰 Tuco's Blackjack Table
Who's brave enough to join? (2/8 players)

Players:
👤 BlondieCoder (Host)
👤 AngelEyes123

[Join 🃏] [Start 🎲]

# Game In Progress
🎰 Tuco's Blackjack Table

Dealer's Hand:
♠️K ⬛ (Face down)

BlondieCoder's Hand (Blackjack! 🎉):
♥️A ♣️K (Value: 21)

AngelEyes123's Hand (Playing):
♦️J ♠️4 (Value: 14)

[Hit 🎯] [Stand 🛑]

# Game Over
🎰 Tuco's Blackjack Table - Game Over!

Dealer's Hand:
♠️K ♥️8 (Value: 18)

BlondieCoder's Hand (Bust 💥):
♥️10 ♣️7 ♠️5 (Value: 22)

AngelEyes123's Hand (Winner! 🍻):
♦️J ♠️4 ♣️5 (Value: 19)

[Play Again 🔄]
```

#### Card Representation
```go
// pkg/discord/messages/cards.go
var (
    // Suit emojis
    SuitSpades   = "♠️"
    SuitHearts   = "♥️"
    SuitClubs    = "♣️"
    SuitDiamonds = "♦️"
    
    // Special cards
    CardFaceDown = "⬛"
    
    // Game states
    StateBlackjack = "🎉"
    StateBust      = "💥"
    StateWin       = "🍻"
    StatePush      = "🤝"
    
    // Actions
    ActionJoin     = "🃏"
    ActionStart    = "🎲"
    ActionHit      = "🎯"
    ActionStand    = "🛑"
    ActionPlayAgain = "🔄"
)

func FormatCard(card *entities.Card) string {
    return fmt.Sprintf("%s%s", getSuitEmoji(card.Suit), card.Rank)
}

func FormatHand(hand *entities.Hand, hidden bool) string {
    if hidden {
        return fmt.Sprintf("%s %s", FormatCard(hand.Cards[0]), CardFaceDown)
    }
    
    cards := make([]string, len(hand.Cards))
    for i, card := range hand.Cards {
        cards[i] = FormatCard(card)
    }
    return strings.Join(cards, " ")
}
```

### Error Handling
Invalid actions trigger ephemeral messages (only visible to the user who clicked):
```
Error Messages:
- "You're already in the game, BLONDIE!"
- "You already stood! No changing your mind!"
- "Only [Host] can start the game!"
- "Game's over! Click Play Again if you dare..."
```

### Design Principles
1. Single Message Updates
   - One persistent message that updates with game state
   - Clear visual progression of the game
   - Maintains chat cleanliness

2. Button-Driven Interaction
   - All game actions performed via buttons
   - Reduces command complexity
   - Improves user experience

3. Flexible Gameplay
   - No strict turn order
   - Players can play at their own pace
   - No timeout mechanisms

4. Simple State Management
   - Clear game phases (Lobby, Playing, Game Over)
   - Minimal state tracking
   - Easy to understand game flow

## Event Flow and Service Integration

### Command to Game Flow
```
Discord Interaction -> Service Layer -> Repository -> Game State -> Message Update
     ↑                                                                  |
     └──────────────────────── Response ─────────────────────────────┘
```

### Button Click Flow Example
```go
// 1. User clicks "Hit" button
ButtonClick("hit") →

// 2. Discord handler processes interaction
func (h *Handler) handleHit(interaction *discordgo.Interaction) {
    // Get game from repository
    game, err := h.games.Get(getGameID(interaction))
    if err != nil {
        return sendError(interaction, "Game not found!")
    }

    // Call service layer
    result, err := h.blackjack.Hit(game.ID, getPlayerID(interaction))
    if err != nil {
        return sendEphemeral(interaction, "Cannot hit: " + err.Error())
    }

    // Update message with new game state
    return updateGameMessage(interaction, result)
}

// 3. Service layer handles game logic
func (s *Service) Hit(gameID GameID, playerID PlayerID) (*GameResult, error) {
    // Get current game state
    game, err := s.games.Get(gameID)
    if err != nil {
        return nil, err
    }

    // Apply game rules
    player, _ := s.players.Get(playerID)
    card := game.Deck.Draw()
    player.Hand.AddCard(card)

    if rules.IsBust(player.Hand) {
        player.Status = entities.PlayerStatusBust
    }

    // Save updated state
    s.games.Update(game)
    s.players.Update(player)

    return &GameResult{
        Game:    game,
        Players: s.getPlayerStates(game),
        Message: "Hit! You drew " + card.String(),
    }, nil
}
```

### Message Update Flow
1. Initial Game Creation
```
/blackjack → CreateGame() → New Game State → Lobby Message
```

2. Player Joins
```
Join Button → JoinGame() → Updated Player List → Updated Lobby Message
```

3. Game Starts
```
Start Button → StartGame() → Deal Cards → Playing Phase Message
```

4. Player Actions
```
Hit/Stand → ProcessAction() → Updated Game State → Updated Game Message
```

5. Game Over
```
Last Action → CheckGameOver() → Final Results → Game Over Message
```

### State Management
1. Discord Message State
   - Each message has a unique ID
   - Game ID stored in message components
   - Player IDs tracked in interaction data

2. Game State Updates
   - Service layer updates game state
   - Repository persists changes
   - Message updates reflect current state

3. Error Handling
   - Invalid actions caught at service layer
   - Ephemeral messages for user errors
   - Game state remains consistent

### Key Integration Points
1. Discord to Service
   - Button interactions map to service methods
   - Interaction data contains all needed IDs
   - Error responses handled uniformly

2. Service to Repository
   - CRUD operations for game state
   - Atomic updates for consistency
   - Error propagation to UI

3. State to Message
   - Game state determines message content
   - Button availability based on game phase
   - Clear feedback for all actions

## Development Phases

1. Phase 1 (Current)
   - Basic bot implementation
   - Single game implementation (Blackjack)
   - Core game mechanics
   - In-memory state management

2. Phase 2
   - Money/wallet management
   - Betting system
   - Enhanced character interactions

3. Future Phases
   - Additional card games
   - Enhanced betting features
   - Extended character development
   - Optional: Persistent storage implementation

## Bot Setup

The bot uses environment variables for configuration:

```env
# .env
DISCORD_TOKEN=your_bot_token
DISCORD_APP_ID=your_app_id
DISCORD_GUILD_ID=your_guild_id  # Optional: for guild-specific commands
```

Initial setup focuses on:
1. Loading environment configuration
2. Establishing Discord connection
3. Basic session management

Commands and game functionality will be added in subsequent phases.
