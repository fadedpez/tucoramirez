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

## Recent Refactoring (March 2025)

### Wallet Integration with Blackjack Payouts

We recently refactored the payout system in the Blackjack game to improve the separation of concerns between the service layer and the Discord handler layer.

#### Changes Made

1. **Service Layer Responsibility**
   - Moved wallet management logic from the Discord handler to the game service layer
   - Created a `GetPlayerWallets` method in the Game struct to encapsulate wallet collection logic
   - Added a `CompleteGameWithPayouts` method to handle game completion and payout processing in one place

2. **Improved Separation of Concerns**
   - Discord handlers now only call service methods and display results
   - All wallet operations (checking balances, processing payouts) happen in the service layer
   - Removed direct wallet manipulation from the Discord layer

3. **Test Improvements**
   - Fixed the `MockWalletService` implementation by adding missing methods
   - Re-enabled previously disabled tests for payout processing

#### Architecture Decisions

The refactoring follows these key principles:

1. **Clean Separation**
   - Game logic (including payouts) belongs in the service layer
   - Discord layer only handles UI and user interaction

2. **Minimal Changes**
   - Made targeted changes only where needed
   - Preserved existing code structure and patterns

3. **Automatic Payout Processing**
   - Payouts are now processed automatically when a game transitions to the complete state
   - This ensures consistent behavior regardless of how the game ends

```go
// Example of the new CompleteGameWithPayouts method
func (g *Game) CompleteGameWithPayouts(ctx context.Context, walletService WalletService) error {
    // Ensure the game is in complete state
    if g.State != entities.StateComplete {
        g.State = entities.StateComplete
    }

    // Process payouts if they haven't been processed yet
    if !g.PayoutsProcessed {
        log.Printf("Processing payouts for completed game in channel %s", g.ChannelID)
        return g.ProcessPayoutsWithWalletUpdates(ctx, walletService)
    }

    return nil
}
```

This approach ensures that the game service maintains full control over the game state and payout processing, while the Discord layer focuses solely on user interaction and display.

## Next Steps: Special Betting Features Implementation

### 1. Entities Layer Updates

#### 1.1 Keep the entities layer generic and add extension interface
```go
// In entities/gamestate.go - No new game state constants
const (
    StateWaiting   Status = "WAITING"
    StateBetting   Status = "BETTING"
    StateDealing   Status = "DEALING"
    StatePlaying   Status = "PLAYING"
    StateDealer    Status = "DEALER"
    StateComplete  Status = "COMPLETE"
)

// Add GameStateExtender interface for game-specific states
type GameStateExtender interface {
    // GetExtendedStates returns additional game-specific states
    GetExtendedStates() []entities.GameState
    
    // IsValidStateTransition checks if a transition between states is valid
    IsValidStateTransition(from, to entities.GameState) bool
}

// Update Hand struct with generic metadata field
type Hand struct {
    // Existing fields...
    Cards        []Card
    Status       Status
    Score        int
    
    // Add a generic metadata field for game-specific extensions
    Metadata     map[string]interface{} // For game-specific data like special bets
}
```

#### 1.2 Update PlayerResult Struct (entities/gamestate.go)
```go
type PlayerResult struct {
    // Existing fields...
    PlayerID   string
    Result     Result
    Payout     int64
    
    // Add generic metadata for special bet results
    Metadata   map[string]interface{}
}
```

### 2. Service Layer Implementation

#### 2.1 Define Blackjack-Specific Constants and State Extender (services/blackjack/rules.go)

```go
// Game states
const (
    StateSpecialBets Status = "SPECIAL_BETS"
)

// Implement GameStateExtender for Blackjack
type BlackjackStateExtender struct{}

func (b *BlackjackStateExtender) GetExtendedStates() []entities.GameState {
    return []entities.GameState{StateSpecialBets}
}

func (b *BlackjackStateExtender) IsValidStateTransition(from, to entities.GameState) bool {
    // Define valid transitions including special bets
    switch from {
    case entities.StateDealing:
        return to == StateSpecialBets || to == entities.StatePlaying
    case StateSpecialBets:
        return to == entities.StatePlaying
    // Other transitions...
    case entities.StateBetting:
        return to == entities.StateDealing
    case entities.StatePlaying:
        return to == entities.StateDealer
    case entities.StateDealer:
        return to == entities.StateComplete
    }
    return false
}

// Add helper method to Game struct for validating transitions
func (g *Game) validateStateTransition(to entities.GameState) error {
    stateExtender := &BlackjackStateExtender{}
    
    if !stateExtender.IsValidStateTransition(g.State, to) {
        return ErrInvalidStateTransition
    }
    
    return nil
}

// Metadata keys for hand
const (
    MetaKeyDoubledDown = "doubled_down"
    MetaKeyDoubleDownBet = "double_down_bet"
    MetaKeySplit = "is_split"
    MetaKeySplitHandID = "split_hand_id"
    MetaKeyParentHandID = "parent_hand_id"
    MetaKeyInsurance = "has_insurance"
    MetaKeyInsuranceBet = "insurance_bet"
)

// Metadata keys for player results
const (
    MetaKeyInsurancePayout = "insurance_payout"
)
```

#### 2.2 Metadata Management
- Define metadata keys for special bets (doubled down, split, insurance)
- Create helper functions to get/set metadata values

#### 2.2 Add Helper Methods for Metadata (services/blackjack/hand.go)

```go
// Helper functions to work with metadata
func isHandDoubledDown(hand *entities.Hand) bool {
    if val, ok := hand.Metadata[MetaKeyDoubledDown]; ok {
        if boolVal, ok := val.(bool); ok {
            return boolVal
        }
    }
    return false
}

func setHandDoubledDown(hand *entities.Hand, value bool) {
    if hand.Metadata == nil {
        hand.Metadata = make(map[string]interface{})
    }
    hand.Metadata[MetaKeyDoubledDown] = value
}

func getDoubleDownBet(hand *entities.Hand) int64 {
    if val, ok := hand.Metadata[MetaKeyDoubleDownBet]; ok {
        if intVal, ok := val.(int64); ok {
            return intVal
        }
    }
    return 0
}

func setDoubleDownBet(hand *entities.Hand, amount int64) {
    if hand.Metadata == nil {
        hand.Metadata = make(map[string]interface{})
    }
    hand.Metadata[MetaKeyDoubleDownBet] = amount
}

// Similar helper functions for split hands
func isSplitHand(hand *entities.Hand) bool {
    if val, ok := hand.Metadata[MetaKeySplit]; ok {
        if boolVal, ok := val.(bool); ok {
            return boolVal
        }
    }
    return false
}

func setSplitHand(hand *entities.Hand, value bool) {
    if hand.Metadata == nil {
        hand.Metadata = make(map[string]interface{})
    }
    hand.Metadata[MetaKeySplit] = value
}

func getSplitHandID(hand *entities.Hand) string {
    if val, ok := hand.Metadata[MetaKeySplitHandID]; ok {
        if strVal, ok := val.(string); ok {
            return strVal
        }
    }
    return ""
}

func setSplitHandID(hand *entities.Hand, handID string) {
    if hand.Metadata == nil {
        hand.Metadata = make(map[string]interface{})
    }
    hand.Metadata[MetaKeySplitHandID] = handID
}

func getParentHandID(hand *entities.Hand) string {
    if val, ok := hand.Metadata[MetaKeyParentHandID]; ok {
        if strVal, ok := val.(string); ok {
            return strVal
        }
    }
    return ""
}

func setParentHandID(hand *entities.Hand, handID string) {
    if hand.Metadata == nil {
        hand.Metadata = make(map[string]interface{})
    }
    hand.Metadata[MetaKeyParentHandID] = handID
}

// Helper functions for insurance
func hasInsurance(hand *entities.Hand) bool {
    if val, ok := hand.Metadata[MetaKeyInsurance]; ok {
        if boolVal, ok := val.(bool); ok {
            return boolVal
        }
    }
    return false
}

func setInsurance(hand *entities.Hand, value bool) {
    if hand.Metadata == nil {
        hand.Metadata = make(map[string]interface{})
    }
    hand.Metadata[MetaKeyInsurance] = value
}

func getInsuranceBet(hand *entities.Hand) int64 {
    if val, ok := hand.Metadata[MetaKeyInsuranceBet]; ok {
        if intVal, ok := val.(int64); ok {
            return intVal
        }
    }
    return 0
}

func setInsuranceBet(hand *entities.Hand, amount int64) {
    if hand.Metadata == nil {
        hand.Metadata = make(map[string]interface{})
    }
    hand.Metadata[MetaKeyInsuranceBet] = amount
}
```

#### 2.3 Update State Transition Methods (services/blackjack/game.go)

```go
// Define a new error for invalid state transitions
var ErrInvalidStateTransition = errors.New("invalid state transition")

// Update existing transition methods to use validation
func (g *Game) StartPlaying() error {
    // Validate the transition
    if err := g.validateStateTransition(entities.StatePlaying); err != nil {
        return err
    }
    
    // Set new state (existing code)
    g.State = entities.StatePlaying
    g.CurrentTurn = 0
    
    return nil
}

// Add new transition methods for special bets
func (g *Game) TransitionToSpecialBets() error {
    // Validate the transition
    if err := g.validateStateTransition(StateSpecialBets); err != nil {
        return err
    }
    
    // Set new state
    g.State = StateSpecialBets
    g.CurrentTurn = 0 // Start with the first player
    
    return nil
}

func (g *Game) TransitionFromSpecialBets() error {
    // Validate the transition
    if err := g.validateStateTransition(entities.StatePlaying); err != nil {
        return err
    }
    
    // Set new state
    g.State = entities.StatePlaying
    g.CurrentTurn = 0 // Start with the first player
    
    return nil
}
```

#### 2.4 Add Special Bets Methods with Loan Functionality (services/blackjack/game.go)

```go
// Helper function to handle funds removal with automatic loan if needed
func (g *Game) handleFundsRemoval(ctx context.Context, playerID string, amount int64, walletService wallet.Service) error {
    // First try to remove funds normally
    err := walletService.RemoveFunds(ctx, playerID, amount)
    if err != nil {
        // If error is insufficient funds, try to take a loan
        if errors.Is(err, wallet.ErrInsufficientFunds) {
            // Get current balance
            balance, err := walletService.GetBalance(ctx, playerID)
            if err != nil {
                return err
            }
            
            // Calculate how much we need to loan
            shortfall := amount - balance
            
            // Take a $100 loan
            const loanAmount int64 = 100
            err = walletService.AddLoan(ctx, playerID, loanAmount)
            if err != nil {
                return err
            }
            
            // Now try to remove funds again
            return walletService.RemoveFunds(ctx, playerID, amount)
        }
        return err
    }
    return nil
}

// Check eligibility for special bets
func (g *Game) IsEligibleForDoubleDown(playerID string) bool {
    // Check if player has exactly 2 cards and hasn't already acted
    hand, exists := g.getPlayerHand(playerID)
    if !exists || len(hand.Cards) != 2 || isHandDoubledDown(&hand) {
        return false
    }
    return true
}

func (g *Game) IsEligibleForSplit(playerID string) bool {
    // Check if player has exactly 2 cards of the same rank
    hand, exists := g.getPlayerHand(playerID)
    if !exists || len(hand.Cards) != 2 {
        return false
    }
    return hand.Cards[0].Rank == hand.Cards[1].Rank
}

func (g *Game) IsEligibleForInsurance() bool {
    // Check if dealer's up card is an Ace
    if len(g.DealerHand.Cards) < 1 {
        return false
    }
    return g.DealerHand.Cards[0].Rank == entities.RankAce
}

// Special bet actions
func (g *Game) DoubleDown(ctx context.Context, playerID string, walletService wallet.Service) error {
    // 1. Check eligibility
    if !g.IsEligibleForDoubleDown(playerID) {
        return errors.New("player not eligible for double down")
    }
    
    // 2. Get player's current bet
    hand, _ := g.getPlayerHand(playerID)
    currentBet := g.Bets[playerID]
    
    // 3. Remove additional bet amount from wallet with loan if needed
    err := g.handleFundsRemoval(ctx, playerID, currentBet, walletService)
    if err != nil {
        return err
    }
    
    // 4. Mark hand as doubled down and store the additional bet
    setHandDoubledDown(&hand, true)
    setDoubleDownBet(&hand, currentBet)
    
    // 5. Deal exactly one more card
    card, err := g.Deck.Draw()
    if err != nil {
        return err
    }
    hand.Cards = append(hand.Cards, card)
    
    // 6. Update hand score
    hand.Score = g.calculateHandScore(hand.Cards)
    
    // 7. Update the hand in the game
    g.updatePlayerHand(playerID, hand)
    
    // 8. Automatically stand (player's turn is over)
    return g.Stand(playerID)
}

func (g *Game) SplitHand(ctx context.Context, playerID string, walletService wallet.Service) error {
    // 1. Check eligibility
    if !g.IsEligibleForSplit(playerID) {
        return errors.New("player not eligible for split")
    }
    
    // 2. Get player's current bet
    hand, _ := g.getPlayerHand(playerID)
    currentBet := g.Bets[playerID]
    
    // 3. Remove additional bet amount from wallet with loan if needed
    err := g.handleFundsRemoval(ctx, playerID, currentBet, walletService)
    if err != nil {
        return err
    }
    
    // 4. Create a new hand with one card from original hand
    newHandID := uuid.New().String()
    newHand := entities.Hand{
        Cards: []entities.Card{hand.Cards[1]},
        Status: hand.Status,
        Metadata: make(map[string]interface{}),
    }
    
    // Update original hand
    hand.Cards = []entities.Card{hand.Cards[0]}
    
    // 5. Deal one new card to each hand
    card1, err := g.Deck.Draw()
    if err != nil {
        return err
    }
    hand.Cards = append(hand.Cards, card1)
    
    card2, err := g.Deck.Draw()
    if err != nil {
        return err
    }
    newHand.Cards = append(newHand.Cards, card2)
    
    // 6. Mark both hands as split and link them
    setSplitHand(&hand, true)
    setSplitHandID(&hand, newHandID)
    
    setSplitHand(&newHand, true)
    setParentHandID(&newHand, playerID)
    
    // 7. Update scores
    hand.Score = g.calculateHandScore(hand.Cards)
    newHand.Score = g.calculateHandScore(newHand.Cards)
    
    // 8. Update the hands in the game
    g.updatePlayerHand(playerID, hand)
    g.PlayerHands[newHandID] = newHand
    g.Bets[newHandID] = currentBet
    
    // 9. Add the new hand to the turn order
    // Insert the new hand right after the current player
    currentIndex := -1
    for i, id := range g.TurnOrder {
        if id == playerID {
            currentIndex = i
            break
        }
    }
    
    if currentIndex >= 0 {
        newTurnOrder := make([]string, 0, len(g.TurnOrder)+1)
        newTurnOrder = append(newTurnOrder, g.TurnOrder[:currentIndex+1]...)
        newTurnOrder = append(newTurnOrder, newHandID)
        newTurnOrder = append(newTurnOrder, g.TurnOrder[currentIndex+1:]...)
        g.TurnOrder = newTurnOrder
    }
    
    return nil
}

func (g *Game) PlaceInsurance(ctx context.Context, playerID string, walletService wallet.Service) error {
    // 1. Check eligibility
    if !g.IsEligibleForInsurance() {
        return errors.New("insurance not available")
    }
    
    // 2. Calculate insurance amount (half of original bet)
    currentBet := g.Bets[playerID]
    insuranceBet := currentBet / 2
    
    // 3. Remove insurance amount from wallet with loan if needed
    err := g.handleFundsRemoval(ctx, playerID, insuranceBet, walletService)
    if err != nil {
        return err
    }
    
    // 4. Mark hand as having insurance
    hand, exists := g.getPlayerHand(playerID)
    if !exists {
        return errors.New("player hand not found")
    }
    
    setInsurance(&hand, true)
    setInsuranceBet(&hand, insuranceBet)
    
    // 5. Update the hand in the game
    g.updatePlayerHand(playerID, hand)
    
    // 6. Advance to next player's turn
    return g.AdvanceSpecialBetsTurn()
}

func (g *Game) DeclineSpecialBet(playerID string) error {
    // Allow player to decline special bet and move to next player
    return g.AdvanceSpecialBetsTurn()
}

// Turn management for special bets
func (g *Game) GetCurrentSpecialBetsPlayerID() (string, error) {
    // Get the current player for special bets phase
    if g.State != StateSpecialBets {
        return "", errors.New("game not in special bets phase")
    }
    
    if g.CurrentTurn >= len(g.TurnOrder) {
        return "", errors.New("no more players for special bets")
    }
    
    return g.TurnOrder[g.CurrentTurn], nil
}

func (g *Game) AdvanceSpecialBetsTurn() error {
    // Move to next player for special bets
    if g.State != StateSpecialBets {
        return errors.New("game not in special bets phase")
    }
    
    g.CurrentTurn++
    
    // If all players have had a chance, transition to PLAYING state
    if g.CurrentTurn >= len(g.TurnOrder) {
        return g.TransitionFromSpecialBets()
    }
    
    return nil
}

// Helper to check if any player is eligible for special bets
func (g *Game) IsAnyPlayerEligibleForSpecialBets() bool {
    for _, playerID := range g.TurnOrder {
        if g.IsEligibleForDoubleDown(playerID) || g.IsEligibleForSplit(playerID) {
            return true
        }
    }
    return false
}
```

### 3. Discord Layer Updates

#### 3.1 Update UI to Show Special Betting Options

```go
// In discord/handlers.go

// Update the updateBettingUI method to include special betting options
func (b *Bot) updateBettingUI(ctx context.Context, channelID string, gameID string) error {
    // Get the game
    game, exists := b.games[gameID]
    if !exists {
        return fmt.Errorf("game not found")
    }
    
    // Get UI info
    uiInfo, err := game.GetGameUIInfo(ctx, b.walletService)
    if err != nil {
        return err
    }
    
    // Create message components based on game state
    var components []discordgo.MessageComponent
    
    // Add special betting buttons when in SPECIAL_BETS state
    if game.State == blackjack.StateSpecialBets {
        currentPlayerID, err := game.GetCurrentSpecialBetsPlayerID()
        if err != nil {
            return err
        }
        
        // Add double down button if eligible
        if game.IsEligibleForDoubleDown(currentPlayerID) {
            components = append(components, discordgo.Button{
                Label:    "Double Down",
                Style:    discordgo.PrimaryButton,
                CustomID: fmt.Sprintf("blackjack:doubledown:%s", gameID),
                Emoji: discordgo.ComponentEmoji{
                    Name: "💰",
                },
            })
        }
        
        // Add split button if eligible
        if game.IsEligibleForSplit(currentPlayerID) {
            components = append(components, discordgo.Button{
                Label:    "Split Hand",
                Style:    discordgo.PrimaryButton,
                CustomID: fmt.Sprintf("blackjack:split:%s", gameID),
                Emoji: discordgo.ComponentEmoji{
                    Name: "✂️",
                },
            })
        }
        
        // Add insurance button if eligible
        if game.IsEligibleForInsurance() {
            components = append(components, discordgo.Button{
                Label:    "Insurance",
                Style:    discordgo.PrimaryButton,
                CustomID: fmt.Sprintf("blackjack:insurance:%s", gameID),
                Emoji: discordgo.ComponentEmoji{
                    Name: "🛡️",
                },
            })
        }
        
        // Always add decline button
        components = append(components, discordgo.Button{
            Label:    "Decline",
            Style:    discordgo.SecondaryButton,
            CustomID: fmt.Sprintf("blackjack:declinespecial:%s", gameID),
        })
    }
    
    // ... existing UI components for other game states
    
    // Create action row with components
    actionRows := []discordgo.ActionsRow{
        {
            Components: components,
        },
    }
    
    // Update the message
    _, err = b.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
        Channel:    channelID,
        ID:         game.MessageID,
        Content:    b.createGameMessage(game, uiInfo),
        Components: actionRows,
    })
    
    return err
}
```

#### 3.2 Add Button Handlers for Special Betting Actions

```go
// In discord/handlers.go

// Register handlers for special betting buttons
func (b *Bot) registerHandlers() {
    // ... existing handlers
    
    // Special betting handlers
    b.session.AddHandler(b.handleDoubleDownButton)
    b.session.AddHandler(b.handleSplitButton)
    b.session.AddHandler(b.handleInsuranceButton)
    b.session.AddHandler(b.handleDeclineSpecialButton)
}

// Handler for double down button
func (b *Bot) handleDoubleDownButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // Check if this is a button interaction
    if i.Type != discordgo.InteractionMessageComponent {
        return
    }
    
    // Parse the custom ID
    parts := strings.Split(i.MessageComponentData().CustomID, ":")
    if len(parts) != 3 || parts[0] != "blackjack" || parts[1] != "doubledown" {
        return
    }
    
    // Get the game ID
    gameID := parts[2]
    game, exists := b.games[gameID]
    if !exists {
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: "Game not found.",
                Flags:   discordgo.MessageFlagsEphemeral,
            },
        })
        return
    }
    
    // Check if it's the player's turn
    currentPlayerID, err := game.GetCurrentSpecialBetsPlayerID()
    if err != nil || currentPlayerID != i.Member.User.ID {
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: "It's not your turn to make a special bet.",
                Flags:   discordgo.MessageFlagsEphemeral,
            },
        })
        return
    }
    
    // Acknowledge the interaction
    s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Flags: discordgo.MessageFlagsEphemeral,
        },
    })
    
    // Execute the double down action
    ctx := context.Background()
    err = game.DoubleDown(ctx, i.Member.User.ID, b.walletService)
    
    // Handle the result
    if err != nil {
        s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
            Content: fmt.Sprintf("Error doubling down: %s", err.Error()),
        })
        return
    }
    
    // Update the UI
    err = b.updateBettingUI(ctx, i.ChannelID, gameID)
    if err != nil {
        s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
            Content: fmt.Sprintf("Error updating UI: %s", err.Error()),
        })
        return
    }
    
    s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
        Content: "You doubled down! Tuco likes your style, amigo!",
    })
}

// Similar handlers for split, insurance, and decline special bet
// ...
```

#### 3.3 Display Split Hands in the UI

```go
// In discord/handlers.go

// Update the createGameMessage function to handle split hands
func (b *Bot) createGameMessage(game *blackjack.Game, uiInfo *blackjack.GameUIInfo) string {
    var message strings.Builder
    
    // ... existing message building code
    
    // Handle split hands in the display
    for _, playerInfo := range uiInfo.AllPlayersInfo {
        hand, exists := game.Players[playerInfo.PlayerID]
        if !exists {
            continue
        }
        
        // Check if this is a split hand
        isSplit := blackjack.IsSplitHand(hand)
        parentID := blackjack.GetParentHandID(hand)
        
        // Format the player name differently for split hands
        playerName := fmt.Sprintf("<@%s>", playerInfo.PlayerID)
        if isSplit {
            playerName = fmt.Sprintf("<@%s>'s Split Hand", parentID)
        }
        
        // Add special bet indicators
        var specialBets []string
        if blackjack.IsHandDoubledDown(hand) {
            specialBets = append(specialBets, "Doubled Down")
        }
        if blackjack.HasInsurance(hand) {
            specialBets = append(specialBets, "Insured")
        }
        
        specialBetsText := ""
        if len(specialBets) > 0 {
            specialBetsText = fmt.Sprintf(" (%s)", strings.Join(specialBets, ", "))
        }
        
        // Add the player's hand to the message
        message.WriteString(fmt.Sprintf("%s%s: %s\n", 
            playerName, 
            specialBetsText,
            formatCards(hand.Cards),
        ))
    }
    
    return message.String()
}
```

## Testing Strategy

#### 4.1 Unit Tests for Eligibility Checks

```go
// In blackjack/game_test.go

func TestIsEligibleForDoubleDown(t *testing.T) {
    // Create a new game
    game := NewGame("test-channel", nil)
    
    // Add a player
    playerID := "test-player"
    game.AddPlayer(playerID)
    
    // Test with no cards (should be ineligible)
    if game.IsEligibleForDoubleDown(playerID) {
        t.Error("Player should not be eligible for double down with no cards")
    }
    
    // Add two cards to the player's hand
    hand, _ := game.getPlayerHand(playerID)
    hand.Cards = []entities.Card{
        {Rank: entities.RankAce, Suit: entities.SuitSpades},
        {Rank: entities.Rank10, Suit: entities.SuitHearts},
    }
    game.updatePlayerHand(playerID, hand)
    
    // Test with two cards (should be eligible)
    if !game.IsEligibleForDoubleDown(playerID) {
        t.Error("Player should be eligible for double down with two cards")
    }
    
    // Mark hand as doubled down
    setHandDoubledDown(&hand, true)
    game.updatePlayerHand(playerID, hand)
    
    // Test with already doubled down hand (should be ineligible)
    if game.IsEligibleForDoubleDown(playerID) {
        t.Error("Player should not be eligible for double down when already doubled down")
    }
    
    // Add a third card
    hand.Cards = append(hand.Cards, entities.Card{Rank: entities.Rank5, Suit: entities.SuitClubs})
    setHandDoubledDown(&hand, false) // Reset doubled down flag
    game.updatePlayerHand(playerID, hand)
    
    // Test with three cards (should be ineligible)
    if game.IsEligibleForDoubleDown(playerID) {
        t.Error("Player should not be eligible for double down with three cards")
    }
}

func TestIsEligibleForSplit(t *testing.T) {
    // Create a new game
    game := NewGame("test-channel", nil)
    
    // Add a player
    playerID := "test-player"
    game.AddPlayer(playerID)
    
    // Test with no cards (should be ineligible)
    if game.IsEligibleForSplit(playerID) {
        t.Error("Player should not be eligible for split with no cards")
    }
    
    // Add two cards of different ranks
    hand, _ := game.getPlayerHand(playerID)
    hand.Cards = []entities.Card{
        {Rank: entities.RankAce, Suit: entities.SuitSpades},
        {Rank: entities.Rank10, Suit: entities.SuitHearts},
    }
    game.updatePlayerHand(playerID, hand)
    
    // Test with different ranks (should be ineligible)
    if game.IsEligibleForSplit(playerID) {
        t.Error("Player should not be eligible for split with different ranked cards")
    }
    
    // Add two cards of the same rank
    hand.Cards = []entities.Card{
        {Rank: entities.Rank8, Suit: entities.SuitSpades},
        {Rank: entities.Rank8, Suit: entities.SuitHearts},
    }
    game.updatePlayerHand(playerID, hand)
    
    // Test with same ranks (should be eligible)
    if !game.IsEligibleForSplit(playerID) {
        t.Error("Player should be eligible for split with same ranked cards")
    }
    
    // Add a third card
    hand.Cards = append(hand.Cards, entities.Card{Rank: entities.Rank5, Suit: entities.SuitClubs})
    game.updatePlayerHand(playerID, hand)
    
    // Test with three cards (should be ineligible)
    if game.IsEligibleForSplit(playerID) {
        t.Error("Player should not be eligible for split with three cards")
    }
}

func TestIsEligibleForInsurance(t *testing.T) {
    // Create a new game
    game := NewGame("test-channel", nil)
    
    // Test with no dealer cards (should be ineligible)
    if game.IsEligibleForInsurance() {
        t.Error("Insurance should not be available with no dealer cards")
    }
    
    // Add a non-Ace as dealer's up card
    game.DealerHand.Cards = []entities.Card{
        {Rank: entities.Rank10, Suit: entities.SuitHearts},
    }
    
    // Test with non-Ace (should be ineligible)
    if game.IsEligibleForInsurance() {
        t.Error("Insurance should not be available when dealer's up card is not an Ace")
    }
    
    // Change dealer's up card to an Ace
    game.DealerHand.Cards = []entities.Card{
        {Rank: entities.RankAce, Suit: entities.SuitSpades},
    }
    
    // Test with Ace (should be eligible)
    if !game.IsEligibleForInsurance() {
        t.Error("Insurance should be available when dealer's up card is an Ace")
    }
}
```

#### 4.2 Integration Tests for State Transitions

```go
// In blackjack/game_test.go

func TestSpecialBetsStateTransitions(t *testing.T) {
    // Create a new game
    game := NewGame("test-channel", nil)
    
    // Add players
    game.AddPlayer("player1")
    game.AddPlayer("player2")
    
    // Set up player order
    game.PlayerOrder = []string{"player1", "player2"}
    
    // Set initial state to DEALING
    game.State = entities.StateDealing
    
    // Test transition to SPECIAL_BETS
    err := game.TransitionToSpecialBets()
    if err != nil {
        t.Errorf("Failed to transition to SPECIAL_BETS: %v", err)
    }
    if game.State != StateSpecialBets {
        t.Errorf("Game state should be SPECIAL_BETS, got %s", game.State)
    }
    if game.CurrentTurn != 0 {
        t.Errorf("Current turn should be 0, got %d", game.CurrentTurn)
    }
    
    // Test advancing turns
    err = game.AdvanceSpecialBetsTurn()
    if err != nil {
        t.Errorf("Failed to advance special bets turn: %v", err)
    }
    if game.CurrentTurn != 1 {
        t.Errorf("Current turn should be 1, got %d", game.CurrentTurn)
    }
    
    // Test transition to PLAYING after all players have had their turn
    err = game.AdvanceSpecialBetsTurn()
    if err != nil {
        t.Errorf("Failed to transition from SPECIAL_BETS to PLAYING: %v", err)
    }
    if game.State != entities.StatePlaying {
        t.Errorf("Game state should be PLAYING, got %s", game.State)
    }
    if game.CurrentTurn != 0 {
        t.Errorf("Current turn should be reset to 0, got %d", game.CurrentTurn)
    }
    
    // Test invalid transitions
    game.State = entities.StatePlaying
    err = game.TransitionToSpecialBets()
    if err == nil {
        t.Error("Should not be able to transition from PLAYING to SPECIAL_BETS")
    }
}
```

#### 4.3 End-to-End Tests for Special Betting Scenarios

```go
// In blackjack/game_test.go

func TestDoubleDownEndToEnd(t *testing.T) {
    // Create a mock wallet service
    mockWallet := &MockWalletService{}
    
    // Create a new game
    game := NewGame("test-channel", nil)
    
    // Add a player
    playerID := "test-player"
    game.AddPlayer(playerID)
    game.PlayerOrder = []string{playerID}
    
    // Set up player's hand with two cards
    hand, _ := game.getPlayerHand(playerID)
    hand.Cards = []entities.Card{
        {Rank: entities.Rank6, Suit: entities.SuitSpades},
        {Rank: entities.Rank5, Suit: entities.SuitHearts},
    }
    hand.Score = 11
    game.updatePlayerHand(playerID, hand)
    
    // Set up player's bet
    game.Bets[playerID] = 10
    
    // Set game state to SPECIAL_BETS
    game.State = StateSpecialBets
    
    // Mock wallet service to simulate sufficient funds
    mockWallet.On("RemoveFunds", mock.Anything, playerID, int64(10)).Return(nil)
    
    // Execute double down
    ctx := context.Background()
    err := game.DoubleDown(ctx, playerID, mockWallet)
    if err != nil {
        t.Errorf("DoubleDown failed: %v", err)
    }
    
    // Verify the hand was updated correctly
    hand, exists := game.getPlayerHand(playerID)
    if !exists {
        t.Fatal("Player hand not found after double down")
    }
    
    // Check that the hand is marked as doubled down
    if !isHandDoubledDown(&hand) {
        t.Error("Hand should be marked as doubled down")
    }
    
    // Check that the double down bet was recorded
    if getDoubleDownBet(&hand) != 10 {
        t.Errorf("Double down bet should be 10, got %d", getDoubleDownBet(&hand))
    }
    
    // Check that a third card was dealt
    if len(hand.Cards) != 3 {
        t.Errorf("Hand should have 3 cards after double down, got %d", len(hand.Cards))
    }
    
    // Verify wallet service was called correctly
    mockWallet.AssertExpectations(t)
}

func TestSplitHandEndToEnd(t *testing.T) {
    // Similar to TestDoubleDownEndToEnd but for split hand functionality
    // ...
}

func TestInsuranceEndToEnd(t *testing.T) {
    // Similar to TestDoubleDownEndToEnd but for insurance functionality
    // ...
}
```

#### 4.4 Tests for Loan Functionality During Special Bets

```go
// In blackjack/game_test.go

func TestHandleFundsRemovalWithLoan(t *testing.T) {
    // Create a mock wallet service
    mockWallet := &MockWalletService{}
    
    // Create a new game
    game := NewGame("test-channel", nil)
    
    // Set up test player
    playerID := "test-player"
    
    // Test case 1: Player has sufficient funds
    mockWallet.On("RemoveFunds", mock.Anything, playerID, int64(50)).Return(nil).Once()
    
    ctx := context.Background()
    err := game.handleFundsRemoval(ctx, playerID, 50, mockWallet)
    if err != nil {
        t.Errorf("handleFundsRemoval failed with sufficient funds: %v", err)
    }
    
    // Test case 2: Player has insufficient funds, needs loan
    insufficientErr := wallet.ErrInsufficientFunds
    mockWallet.On("RemoveFunds", mock.Anything, playerID, int64(100)).Return(insufficientErr).Once()
    mockWallet.On("GetBalance", mock.Anything, playerID).Return(int64(30), nil).Once()
    mockWallet.On("AddLoan", mock.Anything, playerID, int64(100)).Return(nil).Once()
    mockWallet.On("RemoveFunds", mock.Anything, playerID, int64(100)).Return(nil).Once()
    
    err = game.handleFundsRemoval(ctx, playerID, 100, mockWallet)
    if err != nil {
        t.Errorf("handleFundsRemoval failed with loan: %v", err)
    }
    
    // Test case 3: Error getting balance
    balanceErr := errors.New("failed to get balance")
    mockWallet.On("RemoveFunds", mock.Anything, playerID, int64(75)).Return(insufficientErr).Once()
    mockWallet.On("GetBalance", mock.Anything, playerID).Return(int64(0), balanceErr).Once()
    
    err = game.handleFundsRemoval(ctx, playerID, 75, mockWallet)
    if err == nil || err.Error() != balanceErr.Error() {
        t.Errorf("Expected error %v, got %v", balanceErr, err)
    }
    
    // Test case 4: Error adding loan
    loanErr := errors.New("failed to add loan")
    mockWallet.On("RemoveFunds", mock.Anything, playerID, int64(80)).Return(insufficientErr).Once()
    mockWallet.On("GetBalance", mock.Anything, playerID).Return(int64(20), nil).Once()
    mockWallet.On("AddLoan", mock.Anything, playerID, int64(100)).Return(loanErr).Once()
    
    err = game.handleFundsRemoval(ctx, playerID, 80, mockWallet)
    if err == nil || err.Error() != loanErr.Error() {
        t.Errorf("Expected error %v, got %v", loanErr, err)
    }
    
    // Verify all expectations were met
    mockWallet.AssertExpectations(t)
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
