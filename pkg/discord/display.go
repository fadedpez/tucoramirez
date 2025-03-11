package discord

import (
	"github.com/fadedpez/tucoramirez/pkg/entities"
)

func formatCards(cards []*entities.Card) string {
	result := ""
	for _, card := range cards {
		result += formatCard(card) + " "
	}
	return result
}

func formatCard(card *entities.Card) string {
	// Map suits to emoji
	suitEmoji := map[entities.Suit]string{
		entities.Hearts:   "♥️",
		entities.Diamonds: "♦️",
		entities.Clubs:    "♣️",
		entities.Spades:   "♠️",
	}

	// Map ranks to emoji
	rankEmoji := map[entities.Rank]string{
		entities.Ace:   "A️⃣",
		entities.Two:   "2️⃣",
		entities.Three: "3️⃣",
		entities.Four:  "4️⃣",
		entities.Five:  "5️⃣",
		entities.Six:   "6️⃣",
		entities.Seven: "7️⃣",
		entities.Eight: "8️⃣",
		entities.Nine:  "9️⃣",
		entities.Ten:   "🔟",
		entities.Jack:  "J️⃣",
		entities.Queen: "Q️⃣",
		entities.King:  "K️⃣",
	}

	return rankEmoji[card.Rank] + suitEmoji[card.Suit]
}
