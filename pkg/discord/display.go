package discord

import (
	"github.com/fadedpez/tucoramirez/pkg/entities"
)

func FormatCards(cards []*entities.Card) string {
	result := ""
	for i, card := range cards {
		result += FormatCard(card)
		if i < len(cards)-1 {
			result += ", "
		}
	}
	return result
}

func FormatCard(card *entities.Card) string {
	// Map suits to emoji
	suitEmoji := map[entities.Suit]string{
		entities.Hearts:   "♥️",
		entities.Diamonds: "♦️",
		entities.Clubs:    "♣️",
		entities.Spades:   "♠️",
	}

	// Map ranks to emoji
	rankEmoji := map[entities.Rank]string{
		entities.Ace:   ":regional_indicator_a:",
		entities.Two:   "2️⃣",
		entities.Three: "3️⃣",
		entities.Four:  "4️⃣",
		entities.Five:  "5️⃣",
		entities.Six:   "6️⃣",
		entities.Seven: "7️⃣",
		entities.Eight: "8️⃣",
		entities.Nine:  "9️⃣",
		entities.Ten:   "🔟",
		entities.Jack:  ":regional_indicator_j:",
		entities.Queen: ":regional_indicator_q:",
		entities.King:  ":regional_indicator_k:",
	}

	return rankEmoji[card.Rank] + suitEmoji[card.Suit]
}
