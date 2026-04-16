package app

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

func buildReport(accountIDs []int64, heroes map[int]string) (string, error) {
	var builder strings.Builder
	for i, accountID := range accountIDs {
		player, err := fetchPlayerProfile(accountID)
		if err != nil {
			return "", err
		}

		matches, err := fetchPlayerMatches(accountID, 50)
		if err != nil {
			return "", err
		}

		if len(matches) > 10 {
			matches = matches[:10]
		}

		writeMatches(&builder, matches, heroes, player.PersonaName, true)
		if i < len(accountIDs)-1 {
			builder.WriteString("\n\n")
		}
	}
	return builder.String(), nil
}

func buildPlayerTable(matches []recentMatch, heroes map[int]string, playerName string) string {
	var builder strings.Builder
	writeMatches(&builder, matches, heroes, playerName, false)
	return builder.String()
}

func buildRatingTable(accountIDs []int64) (string, error) {
	type ratingEntry struct {
		Name    string
		Winrate float64
		Games   int
	}
	entries := make([]ratingEntry, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		player, err := fetchPlayerProfile(accountID)
		if err != nil {
			return "", err
		}
		matches, err := fetchPlayerMatches(accountID, 50)
		if err != nil {
			return "", err
		}
		winrate, games := calcWinrateWithCount(matches, 50)
		name := player.PersonaName
		if name == "" {
			name = "неизвестный"
		}
		entries = append(entries, ratingEntry{
			Name:    name,
			Winrate: winrate,
			Games:   games,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Winrate == entries[j].Winrate {
			return entries[i].Games > entries[j].Games
		}
		return entries[i].Winrate > entries[j].Winrate
	})

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%-3s  %-16s  %-9s  %-5s\n", "№", "Игрок", "Winrate", "Игр"))
	for i, e := range entries {
		rank := fmt.Sprintf("%d", i+1)
		switch i {
		case 0:
			rank = "🥇"
		case 1:
			rank = "🥈"
		case 2:
			rank = "🥉"
		}
		builder.WriteString(fmt.Sprintf("%-3s  %-16s  %7.1f%%  %-5d\n", rank, trimTo(e.Name, 16), e.Winrate, e.Games))
	}
	return builder.String(), nil
}

func buildBestFriendsTable(accountIDs []int64, limit int) (string, error) {
	type bestFriendEntry struct {
		Player  string
		Friend  string
		Winrate float64
		Games   int
	}
	allowedFriends := make(map[int64]struct{}, len(accountIDs))
	for _, id := range accountIDs {
		allowedFriends[id] = struct{}{}
	}
	nameByID := make(map[int64]string, len(accountIDs))
	for _, id := range accountIDs {
		player, err := fetchPlayerProfile(id)
		if err != nil {
			return "", err
		}
		nameByID[id] = fallbackName(player.PersonaName)
	}
	entries := make([]bestFriendEntry, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		playerName := nameByID[accountID]
		best := bestFriendEntry{
			Player: playerName,
			Friend: "нет данных",
		}
		for _, friendID := range accountIDs {
			if friendID == accountID {
				continue
			}
			if _, ok := allowedFriends[friendID]; !ok {
				continue
			}
			matches, err := fetchMatchesWith(accountID, friendID, limit)
			if err != nil {
				return "", err
			}
			winrate, games := calcWinrateFromMatches(matches)
			if games == 0 {
				continue
			}
			if winrate > best.Winrate || (winrate == best.Winrate && games > best.Games) {
				name := nameByID[friendID]
				if name == "" {
					name = fmt.Sprintf("Account %d", friendID)
				}
				best.Friend = name
				best.Winrate = winrate
				best.Games = games
			}
		}
		entries = append(entries, best)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%-16s  %-16s  %-9s  %-5s\n", "Игрок", "Лучший друг", "Winrate", "Игр"))
	for _, e := range entries {
		builder.WriteString(fmt.Sprintf("%-16s  %-16s  %7.1f%%  %-5d\n", trimTo(e.Player, 16), trimTo(e.Friend, 16), e.Winrate, e.Games))
	}
	return builder.String(), nil
}

func writeMatches(writer io.Writer, matches []recentMatch, heroes map[int]string, playerName string, includeTitle bool) {
	if playerName == "" {
		playerName = "неизвестный"
	}
	if includeTitle {
		fmt.Fprintf(writer, "Последние матчи (%s):\n", playerName)
	}
	fmt.Fprintf(writer, "%-16s  %-12s  %-4s  %-7s  %-6s\n", "Дата", "Герой", "Итог", "K/D/A", "Длит.")
	for _, m := range matches {
		heroName := heroes[m.HeroID]
		if heroName == "" {
			heroName = fmt.Sprintf("Hero #%d", m.HeroID)
		}
		win := matchWin(m)
		result := "❌"
		if win {
			result = "✅"
		}
		start := time.Unix(m.StartTime, 0).Local().Format("2006-01-02 15:04")
		duration := formatDuration(m.Duration)
		kda := fmt.Sprintf("%d/%d/%d", m.Kills, m.Deaths, m.Assists)
		fmt.Fprintf(writer, "%-16s  %-12s  %-4s  %-7s  %-6s\n", start, trimTo(heroName, 12), result, kda, duration)
	}
}

func formatMatchSummary(playerName string, match recentMatch, heroes map[int]string) string {
	heroName := heroes[match.HeroID]
	if heroName == "" {
		heroName = fmt.Sprintf("Hero #%d", match.HeroID)
	}
	result := "❌"
	if matchWin(match) {
		result = "✅"
	}
	duration := formatDuration(match.Duration)
	kda := fmt.Sprintf("%d/%d/%d", match.Kills, match.Deaths, match.Assists)
	return fmt.Sprintf("%s | %s | %s | %s | %s", result, fallbackName(playerName), heroName, kda, duration)
}
