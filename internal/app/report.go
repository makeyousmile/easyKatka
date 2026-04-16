package app

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

type matchNotification struct {
	Text     string
	PhotoURL string
	MatchID  int64
	AccountID int64
}

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

func formatMatchSummary(match recentMatch, heroes map[int]string) string {
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
	return fmt.Sprintf("%s | %s | %s | %s", result, heroName, kda, duration)
}

func buildTestMatchSummary(accountIDs []int64, heroes map[int]string) (matchNotification, error) {
	if len(accountIDs) == 0 {
		return matchNotification{}, fmt.Errorf("нет аккаунтов для тестового сообщения")
	}

	for _, accountID := range accountIDs {
		player, err := fetchPlayerProfile(accountID)
		if err != nil {
			return matchNotification{}, err
		}

		matches, err := fetchRecentMatches(accountID)
		if err != nil {
			return matchNotification{}, err
		}
		if len(matches) == 0 {
			continue
		}

		return matchNotification{
			Text:      formatMatchSummary(matches[0], heroes),
			PhotoURL:  player.AvatarFull,
			MatchID:   matches[0].MatchID,
			AccountID: accountID,
		}, nil
	}

	return matchNotification{}, fmt.Errorf("не найдено ни одного матча для тестового сообщения")
}

func findPlayerInMatch(details matchDetails, accountID int64) *matchDetailsPlayer {
	for i := range details.Players {
		if details.Players[i].AccountID == accountID {
			return &details.Players[i]
		}
	}
	return nil
}

func formatMatchDetailsMessage(details matchDetails, accountID int64, heroes map[int]string, itemNames map[int]string) (string, error) {
	player := findPlayerInMatch(details, accountID)
	if player == nil {
		return "", fmt.Errorf("игрок не найден в деталях матча")
	}

	heroName := heroes[player.HeroID]
	if heroName == "" {
		heroName = fmt.Sprintf("Hero #%d", player.HeroID)
	}

	result := "❌ Поражение"
	if matchWin(recentMatch{PlayerSlot: player.PlayerSlot, RadiantWin: details.RadiantWin}) {
		result = "✅ Победа"
	}

	itemList := collectPlayerItems(*player, itemNames)
	itemsText := "нет данных"
	if len(itemList) > 0 {
		itemsText = strings.Join(itemList, ", ")
	}

	lines := []string{
		fmt.Sprintf("<b>%s</b>", escapeHTML(result)),
		fmt.Sprintf("<b>Игрок:</b> %s", escapeHTML(fallbackName(player.PersonaName))),
		fmt.Sprintf("<b>Герой:</b> %s", escapeHTML(heroName)),
		fmt.Sprintf("<b>K/D/A:</b> <code>%d/%d/%d</code>", player.Kills, player.Deaths, player.Assists),
		fmt.Sprintf("<b>Длительность:</b> <code>%s</code>", formatDuration(details.Duration)),
		fmt.Sprintf("<b>Счёт:</b> <code>%d:%d</code>", details.RadiantScore, details.DireScore),
		fmt.Sprintf("<b>GPM/XPM:</b> <code>%d/%d</code>", player.GPM, player.XPM),
		fmt.Sprintf("<b>LH/DN:</b> <code>%d/%d</code>", player.LastHits, player.Denies),
		fmt.Sprintf("<b>Hero Damage:</b> <code>%d</code>", player.HeroDamage),
		fmt.Sprintf("<b>Tower Damage:</b> <code>%d</code>", player.TowerDamage),
		fmt.Sprintf("<b>Hero Healing:</b> <code>%d</code>", player.HeroHealing),
		fmt.Sprintf("<b>Net Worth:</b> <code>%d</code>", player.NetWorth),
		fmt.Sprintf("<b>Предметы:</b> %s", escapeHTML(itemsText)),
		fmt.Sprintf("<b>Match ID:</b> <code>%d</code>", details.MatchID),
		fmt.Sprintf("<a href=\"https://www.opendota.com/matches/%d\">OpenDota</a>", details.MatchID),
	}
	return strings.Join(lines, "\n"), nil
}

func collectPlayerItems(player matchDetailsPlayer, itemNames map[int]string) []string {
	itemIDs := []int{
		player.Item0, player.Item1, player.Item2, player.Item3, player.Item4, player.Item5,
		player.Backpack0, player.Backpack1, player.Backpack2, player.NeutralItem,
	}
	items := make([]string, 0, len(itemIDs))
	for _, itemID := range itemIDs {
		if itemID <= 0 {
			continue
		}
		if name := strings.TrimSpace(itemNames[itemID]); name != "" {
			items = append(items, name)
			continue
		}
		items = append(items, fmt.Sprintf("Item #%d", itemID))
	}
	return items
}
