package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/pkg/services/statistics"
)

// StatsCommand handles the /stats command for displaying player statistics
type StatsCommand struct {
	statisticsService *statistics.Service
	blackjackCommand  BlackjackCommand // For starting new games
}

// BlackjackCommand interface defines the methods we need from the blackjack command
type BlackjackCommand interface {
	StartGame(ctx context.Context, s *discordgo.Session, channelID, userID string) error
}

// NewStatsCommand creates a new stats command handler
func NewStatsCommand(statisticsService *statistics.Service, blackjackCommand BlackjackCommand) *StatsCommand {
	return &StatsCommand{
		statisticsService: statisticsService,
		blackjackCommand:  blackjackCommand,
	}
}

// Command returns the command definition for the stats command
func (c *StatsCommand) Command() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "stats",
		Description: "View player statistics",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "blackjack",
				Description: "View blackjack statistics",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
		},
	}
}

// respondWithError sends an error message as a response to an interaction
func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// Handle handles the stats command
func (c *StatsCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Get the subcommand
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		respondWithError(s, i, "Please specify a game type (e.g., /stats blackjack)")
		return
	}

	subcmd := options[0].Name
	switch subcmd {
	case "blackjack":
		c.handleBlackjackStats(s, i)
	default:
		respondWithError(s, i, "Unknown game type. Try /stats blackjack")
	}
}

// handleBlackjackStats handles the /stats blackjack subcommand
func (c *StatsCommand) handleBlackjackStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	// Get the first page of the leaderboard
	leaderboard, err := c.statisticsService.GetBlackjackLeaderboard(ctx, 1, 10)
	if err != nil {
		respondWithError(s, i, "Failed to get statistics: "+err.Error())
		return
	}

	// Set initial view type to core stats
	viewType := "core"
	
	// Create the initial response with the specified view type
	embed := c.createLeaderboardEmbed(leaderboard, viewType)
	paginationRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Previous",
				Style:    discordgo.SecondaryButton,
				CustomID: "stats_prev",
				Disabled: leaderboard.CurrentPage <= 1,
				Emoji: &discordgo.ComponentEmoji{
					Name: "⬅️",
				},
			},
			discordgo.Button{
				Label:    "Refresh",
				Style:    discordgo.SecondaryButton,
				CustomID: "stats_refresh",
				Emoji: &discordgo.ComponentEmoji{
					Name: "🔄",
				},
			},
			discordgo.Button{
				Label:    "Next",
				Style:    discordgo.SecondaryButton,
				CustomID: "stats_next",
				Disabled: leaderboard.CurrentPage >= leaderboard.TotalPages,
				Emoji: &discordgo.ComponentEmoji{
					Name: "➡️",
				},
			},
		},
	}
	
	// Determine button styles based on view type
	coreButtonStyle := discordgo.PrimaryButton
	specialButtonStyle := discordgo.SecondaryButton
	
	toggleRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Core Stats",
				Style:    coreButtonStyle,
				CustomID: "stats_view_core",
				Disabled: true, // Initially disabled as we're showing core stats
				Emoji: &discordgo.ComponentEmoji{
					Name: "📊",
				},
			},
			discordgo.Button{
				Label:    "Special Stats",
				Style:    specialButtonStyle,
				CustomID: "stats_view_special",
				Emoji: &discordgo.ComponentEmoji{
					Name: "🎯",
				},
			},
			discordgo.Button{
				Label:    "Play Blackjack",
				Style:    discordgo.SuccessButton,
				CustomID: "stats_play_blackjack",
				Emoji: &discordgo.ComponentEmoji{
					Name: "🃏",
				},
			},
		},
	}

	// Respond with the leaderboard
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{paginationRow, toggleRow},
		},
	})
}

// createLeaderboardEmbed creates an embed for the leaderboard
func (c *StatsCommand) createLeaderboardEmbed(leaderboard *statistics.BlackjackLeaderboard, viewType string) *discordgo.MessageEmbed {
	// Create the title
	title := "🎮 Blackjack Leaderboard 🎮"

	// Create the description
	description := fmt.Sprintf("Showing page %d of %d (%d total players)", 
		leaderboard.CurrentPage, leaderboard.TotalPages, leaderboard.TotalPlayers)

	// Create the fields for each player
	fields := make([]*discordgo.MessageEmbedField, 0, len(leaderboard.Players))

	for _, player := range leaderboard.Players {
		// Create the player name with rank
		rankEmoji := ""
		switch player.Rank {
		case 1:
			rankEmoji = "👑 "
		case 2:
			rankEmoji = "🥈 "
		case 3:
			rankEmoji = "🥉 "
		default:
			rankEmoji = fmt.Sprintf("%d. ", player.Rank)
		}

		// Add special indicators
		specialIndicators := ""
		if player.IsTopWinner && player.Rank != 1 {
			specialIndicators += "💰 "
		}
		if player.IsTopPlayer {
			specialIndicators += "🏆 "
		}

		playerName := fmt.Sprintf("%s%s%s", rankEmoji, player.PlayerID, specialIndicators)

		// Create the player stats
		winRate := player.WinRate * 100
		profitRate := (player.ProfitRate - 1) * 100
		profitRateStr := ""
		if profitRate > 0 {
			profitRateStr = fmt.Sprintf("+%.1f%%", profitRate)
		} else {
			profitRateStr = fmt.Sprintf("%.1f%%", profitRate)
		}

		recordStr := fmt.Sprintf("%dW-%dL-%dP", player.Wins, player.Losses, player.Pushes)
		specialStats := []string{}

		if player.Blackjacks > 0 {
			specialStats = append(specialStats, fmt.Sprintf("%d BJ", player.Blackjacks))
		}
		if player.Busts > 0 {
			specialStats = append(specialStats, fmt.Sprintf("%d Busts", player.Busts))
		}
		if player.Splits > 0 {
			specialStats = append(specialStats, fmt.Sprintf("%d Splits", player.Splits))
		}
		if player.DoubleDowns > 0 {
			specialStats = append(specialStats, fmt.Sprintf("%d DD", player.DoubleDowns))
		}
		// Insurance bets are not tracked in PlayerRank currently

		specialStatsStr := ""
		if len(specialStats) > 0 {
			specialStatsStr = "\n" + strings.Join(specialStats, ", ")
		}

		valueStr := ""
		switch viewType {
		case "core":
			valueStr = fmt.Sprintf(
				"**Games:** %d | **Record:** %s | **Win Rate:** %.1f%%\n**Total Bet:** $%d | **Winnings:** $%d | **ROI:** %s%s",
				player.GamesPlayed, recordStr, winRate, player.TotalBet, player.TotalWinnings, profitRateStr, specialStatsStr,
			)
		case "special":
			valueStr = fmt.Sprintf(
				"**Blackjacks:** %d | **Busts:** %d | **Splits:** %d | **Double Downs:** %d\n**Insurances:** %d",
				player.Blackjacks, player.Busts, player.Splits, player.DoubleDowns, player.Insurances,
			)
		}

		// Add the field
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   playerName,
			Value:  valueStr,
			Inline: false,
		})
	}

	// Add a footer with legend
	footer := "👑 = #1 Winnings | 💰 = Highest Winnings | 🏆 = Most Games Played"

	// Create the embed
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x00ff00,
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: footer,
		},
		Timestamp: leaderboard.LastUpdated.Format(time.RFC3339),
	}
}

// HandleComponentInteraction handles button clicks on the leaderboard
func (c *StatsCommand) HandleComponentInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	data := i.MessageComponentData()
	customID := data.CustomID

	switch customID {
	case "stats_prev", "stats_next", "stats_refresh":
		c.handlePaginationInteraction(s, i, customID)
		return true
	case "stats_view_core", "stats_view_special":
		c.handleViewToggleInteraction(s, i, customID)
		return true
	case "stats_play_blackjack":
		c.handlePlayBlackjackInteraction(s, i)
		return true
	default:
		return false
	}
}

// handlePaginationInteraction handles pagination button clicks
func (c *StatsCommand) handlePaginationInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	// Acknowledge the interaction
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	// Get the current page from the embed description
	embed := i.Message.Embeds[0]
	currentPage := 1
	totalPages := 1

	// Parse the description to get the current page and total pages
	fmt.Sscanf(embed.Description, "Showing page %d of %d", &currentPage, &totalPages)

	// Calculate the new page
	newPage := currentPage
	switch customID {
	case "stats_prev":
		if currentPage > 1 {
			newPage = currentPage - 1
		}
	case "stats_next":
		if currentPage < totalPages {
			newPage = currentPage + 1
		}
	case "stats_refresh":
		// Keep the same page, but refresh the data
	}

	// Determine the current view type by checking which button is disabled
	viewType := "core" // Default to core view
	for _, component := range i.Message.Components {
		if actionRow, ok := component.(*discordgo.ActionsRow); ok {
			for _, btn := range actionRow.Components {
				if button, ok := btn.(*discordgo.Button); ok {
					if button.CustomID == "stats_view_special" && button.Disabled {
						viewType = "special"
						break
					}
				}
			}
		}
	}

	// Get the updated leaderboard
	ctx := context.Background()
	leaderboard, err := c.statisticsService.GetBlackjackLeaderboard(ctx, newPage, 10)
	if err != nil {
		// If there's an error, just keep the current page
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Failed to update leaderboard: " + err.Error(),
		})
		return
	}

	// Create the updated embed and components
	newEmbed := c.createLeaderboardEmbed(leaderboard, viewType)

	// Determine button styles based on view type
	coreButtonStyle := discordgo.SecondaryButton
	specialButtonStyle := discordgo.SecondaryButton
	if viewType == "core" {
		coreButtonStyle = discordgo.PrimaryButton
	} else {
		specialButtonStyle = discordgo.PrimaryButton
	}

	paginationRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Previous",
				Style:    discordgo.SecondaryButton,
				CustomID: "stats_prev",
				Disabled: leaderboard.CurrentPage <= 1,
				Emoji: &discordgo.ComponentEmoji{
					Name: "⬅️",
				},
			},
			discordgo.Button{
				Label:    "Refresh",
				Style:    discordgo.SecondaryButton,
				CustomID: "stats_refresh",
				Emoji: &discordgo.ComponentEmoji{
					Name: "🔄",
				},
			},
			discordgo.Button{
				Label:    "Next",
				Style:    discordgo.SecondaryButton,
				CustomID: "stats_next",
				Disabled: leaderboard.CurrentPage >= leaderboard.TotalPages,
				Emoji: &discordgo.ComponentEmoji{
					Name: "➡️",
				},
			},
		},
	}
	toggleRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Core Stats",
				Style:    coreButtonStyle,
				CustomID: "stats_view_core",
				Disabled: viewType == "core", // Disable the button if we're already showing core stats
				Emoji: &discordgo.ComponentEmoji{
					Name: "📊",
				},
			},
			discordgo.Button{
				Label:    "Special Stats",
				Style:    specialButtonStyle,
				CustomID: "stats_view_special",
				Disabled: viewType == "special", // Disable the button if we're already showing special stats
				Emoji: &discordgo.ComponentEmoji{
					Name: "🎯",
				},
			},
			discordgo.Button{
				Label:    "Play Blackjack",
				Style:    discordgo.SuccessButton,
				CustomID: "stats_play_blackjack",
				Emoji: &discordgo.ComponentEmoji{
					Name: "🃏",
				},
			},
		},
	}

	// Update the message
	_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: i.ChannelID,
		ID:      i.Message.ID,
		Embeds:  &[]*discordgo.MessageEmbed{newEmbed},
		Components: &[]discordgo.MessageComponent{
			paginationRow,
			toggleRow,
		},
	})

	if err != nil {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Failed to update message: " + err.Error(),
		})
	}
}

// handleViewToggleInteraction handles the view toggle button click
func (c *StatsCommand) handleViewToggleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	// Acknowledge the interaction
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	// Get the current page from the embed description
	embed := i.Message.Embeds[0]
	currentPage := 1
	totalPages := 1

	// Parse the description to get the current page and total pages
	fmt.Sscanf(embed.Description, "Showing page %d of %d", &currentPage, &totalPages)

	// Get the updated leaderboard
	ctx := context.Background()
	leaderboard, err := c.statisticsService.GetBlackjackLeaderboard(ctx, currentPage, 10)
	if err != nil {
		// If there's an error, just keep the current page
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Failed to update leaderboard: " + err.Error(),
		})
		return
	}

	// Create the updated embed and components
	viewType := "core"
	if customID == "stats_view_special" {
		viewType = "special"
	}
	newEmbed := c.createLeaderboardEmbed(leaderboard, viewType)
	paginationRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Previous",
				Style:    discordgo.SecondaryButton,
				CustomID: "stats_prev",
				Disabled: leaderboard.CurrentPage <= 1,
				Emoji: &discordgo.ComponentEmoji{
					Name: "⬅️",
				},
			},
			discordgo.Button{
				Label:    "Refresh",
				Style:    discordgo.SecondaryButton,
				CustomID: "stats_refresh",
				Emoji: &discordgo.ComponentEmoji{
					Name: "🔄",
				},
			},
			discordgo.Button{
				Label:    "Next",
				Style:    discordgo.SecondaryButton,
				CustomID: "stats_next",
				Disabled: leaderboard.CurrentPage >= leaderboard.TotalPages,
				Emoji: &discordgo.ComponentEmoji{
					Name: "➡️",
				},
			},
		},
	}
	
	// Determine button styles based on view type
	coreButtonStyle := discordgo.SecondaryButton
	specialButtonStyle := discordgo.SecondaryButton
	if viewType == "core" {
		coreButtonStyle = discordgo.PrimaryButton
	} else {
		specialButtonStyle = discordgo.PrimaryButton
	}
	
	toggleRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Core Stats",
				Style:    coreButtonStyle,
				CustomID: "stats_view_core",
				Disabled: viewType == "core", // Disable the button if we're already showing core stats
				Emoji: &discordgo.ComponentEmoji{
					Name: "📊",
				},
			},
			discordgo.Button{
				Label:    "Special Stats",
				Style:    specialButtonStyle,
				CustomID: "stats_view_special",
				Disabled: viewType == "special", // Disable the button if we're already showing special stats
				Emoji: &discordgo.ComponentEmoji{
					Name: "🎯",
				},
			},
			discordgo.Button{
				Label:    "Play Blackjack",
				Style:    discordgo.SuccessButton,
				CustomID: "stats_play_blackjack",
				Emoji: &discordgo.ComponentEmoji{
					Name: "🃏",
				},
			},
		},
	}

	// Update the message
	_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: i.ChannelID,
		ID:      i.Message.ID,
		Embeds:  &[]*discordgo.MessageEmbed{newEmbed},
		Components: &[]discordgo.MessageComponent{
			paginationRow,
			toggleRow,
		},
	})

	if err != nil {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Failed to update message: " + err.Error(),
		})
	}
}

// handlePlayBlackjackInteraction handles the Play Blackjack button click
func (c *StatsCommand) handlePlayBlackjackInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Acknowledge the interaction
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	// Start a new blackjack game
	ctx := context.Background()
	err := c.blackjackCommand.StartGame(ctx, s, i.ChannelID, i.Member.User.ID)
	if err != nil {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Failed to start blackjack game: " + err.Error(),
		})
		return
	}

	// Delete the stats message to keep the channel clean
	s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
}
