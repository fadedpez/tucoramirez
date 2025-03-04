package cards

import "math/rand"

type Card struct {
	Suit  string
	Value string
}

type Player struct {
	Name  string
	ID    string
	Hand  []Card
	Score int
}

type GameSession struct {
	Players []Player
	Deck    []Card
	Dealer  *Player
	ID      string
}

func CreateDeck() []Card {
	var newDeck []Card
	suits := []string{":hearts:", ":diamonds:", ":clubs:", ":spades:"}
	values := []string{":two:", ":three:", ":four:", ":five:", ":six:", ":seven:", ":eight:", ":nine:", ":keycap_ten:", ":regional_indicator_j:", ":regional_indicator_q:", ":regional_indicator_k:", ":regional_indicator_a:"}

	for _, suit := range suits {
		for _, value := range values {
			newDeck = append(newDeck, Card{Suit: suit, Value: value})
		}
	}

	return newDeck
}

func ShuffleDeck(deck []Card) []Card {
	for i := range deck {
		j := rand.Intn(i + 1)
		deck[i], deck[j] = deck[j], deck[i]
	}

	return deck
}
