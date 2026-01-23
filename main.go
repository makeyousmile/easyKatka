package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	baseURL          = "https://api.opendota.com/api"
	steamID64Offset  = int64(76561197960265728)
	maxUint32        = int64(^uint32(0))
	requestTimeout   = 15 * time.Second
	recentMatchesURL = "/players/%d/recentMatches"
	heroesURL        = "/heroes"
	playerURL        = "/players/%d"
	peersURL         = "/players/%d/peers"
	playerMatchesURL = "/players/%d/matches"

	telegramTokenEnv = "TELEGRAM_BOT_TOKEN"
	telegramChatEnv  = "TELEGRAM_NOTIFY_CHAT_ID"
	telegramBaseURL  = "https://api.telegram.org/bot%s"
	telegramMaxLen   = 3900
)

type hero struct {
	ID            int    `json:"id"`
	LocalizedName string `json:"localized_name"`
}

type recentMatch struct {
	MatchID    int64 `json:"match_id"`
	HeroID     int   `json:"hero_id"`
	Kills      int   `json:"kills"`
	Deaths     int   `json:"deaths"`
	Assists    int   `json:"assists"`
	Duration   int   `json:"duration"`
	StartTime  int64 `json:"start_time"`
	PlayerSlot int   `json:"player_slot"`
	RadiantWin bool  `json:"radiant_win"`
}

type playerProfile struct {
	Profile struct {
		PersonaName string `json:"personaname"`
		AvatarFull  string `json:"avatarfull"`
	} `json:"profile"`
}

type playerProfileData struct {
	PersonaName string
	AvatarFull  string
}

type peerEntry struct {
	AccountID   int64  `json:"account_id"`
	Personaname string `json:"personaname"`
	WithGames   int    `json:"with_games"`
	WithWin     int    `json:"with_win"`
}

type playerMatch struct {
	MatchID    int64 `json:"match_id"`
	PlayerSlot int   `json:"player_slot"`
	RadiantWin bool  `json:"radiant_win"`
}

type telegramUpdate struct {
	UpdateID int              `json:"update_id"`
	Message  *telegramMessage `json:"message"`
}

type telegramMessage struct {
	MessageID int          `json:"message_id"`
	Chat      telegramChat `json:"chat"`
	Text      string       `json:"text"`
}

type telegramChat struct {
	ID int64 `json:"id"`
}

type telegramUpdatesResponse struct {
	OK          bool             `json:"ok"`
	Result      []telegramUpdate `json:"result"`
	Description string           `json:"description"`
}

func main() {
	accountIDs, err := loadAccountIDs("account_id")
	if err != nil {
		exitErr(err)
	}

	heroes, err := fetchHeroes()
	if err != nil {
		exitErr(err)
	}

	telegramToken := strings.TrimSpace(os.Getenv(telegramTokenEnv))
	if telegramToken != "" {
		if notify := telegramNotifier(telegramToken); notify != nil {
			go monitorMatches(accountIDs, heroes, notify)
		}
		if err := runTelegramBot(telegramToken, accountIDs, heroes); err != nil {
			exitErr(err)
		}
		return
	}

	report, err := buildReport(accountIDs, heroes)
	if err != nil {
		exitErr(err)
	}
	fmt.Print(report)

	monitorMatches(accountIDs, heroes, nil)
}

func loadAccountIDs(path string) ([]int64, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read account_id: %w", err)
	}
	lines := strings.Split(string(raw), "\n")
	var ids []int64
	for _, line := range lines {
		value := strings.TrimSpace(line)
		if value == "" {
			continue
		}
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse account_id: %w", err)
		}
		if id > maxUint32 {
			id = id - steamID64Offset
		}
		if id <= 0 {
			return nil, fmt.Errorf("invalid account_id after conversion: %d", id)
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("account_id is empty")
	}
	return ids, nil
}

func fetchHeroes() (map[int]string, error) {
	var heroes []hero
	if err := getJSON(baseURL+heroesURL, &heroes); err != nil {
		return nil, err
	}
	result := make(map[int]string, len(heroes))
	for _, h := range heroes {
		if h.LocalizedName == "" {
			continue
		}
		result[h.ID] = h.LocalizedName
	}
	return result, nil
}

func fetchRecentMatches(accountID int64) ([]recentMatch, error) {
	var matches []recentMatch
	url := fmt.Sprintf(baseURL+recentMatchesURL, accountID)
	if err := getJSON(url, &matches); err != nil {
		return nil, err
	}
	return matches, nil
}

func fetchPlayerProfile(accountID int64) (playerProfileData, error) {
	var player playerProfile
	url := fmt.Sprintf(baseURL+playerURL, accountID)
	if err := getJSON(url, &player); err != nil {
		return playerProfileData{}, err
	}
	return playerProfileData{
		PersonaName: strings.TrimSpace(player.Profile.PersonaName),
		AvatarFull:  strings.TrimSpace(player.Profile.AvatarFull),
	}, nil
}

func fetchPeers(accountID int64) ([]peerEntry, error) {
	var peers []peerEntry
	url := fmt.Sprintf(baseURL+peersURL, accountID)
	if err := getJSON(url, &peers); err != nil {
		return nil, err
	}
	return peers, nil
}

func fetchMatchesWith(accountID int64, includedAccountID int64, limit int) ([]playerMatch, error) {
	var matches []playerMatch
	url := fmt.Sprintf("%s"+playerMatchesURL+"?included_account_id=%d&limit=%d", baseURL, accountID, includedAccountID, limit)
	if err := getJSON(url, &matches); err != nil {
		return nil, err
	}
	return matches, nil
}

func getJSON(url string, out any) error {
	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("request %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("request %s failed: %s: %s", url, resp.Status, strings.TrimSpace(string(body)))
	}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", url, err)
	}
	return nil
}

func buildReport(accountIDs []int64, heroes map[int]string) (string, error) {
	var builder strings.Builder
	for i, accountID := range accountIDs {
		player, err := fetchPlayerProfile(accountID)
		if err != nil {
			return "", err
		}

		matches, err := fetchRecentMatches(accountID)
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
		matches, err := fetchRecentMatches(accountID)
		if err != nil {
			return "", err
		}
		winrate, games := calcWinrateWithCount(matches, 20)
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
	entries := make([]bestFriendEntry, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		player, err := fetchPlayerProfile(accountID)
		if err != nil {
			return "", err
		}
		peers, err := fetchPeers(accountID)
		if err != nil {
			return "", err
		}
		sort.Slice(peers, func(i, j int) bool {
			return peers[i].WithGames > peers[j].WithGames
		})
		if len(peers) > 20 {
			peers = peers[:20]
		}
		best := bestFriendEntry{
			Player: fallbackName(player.PersonaName),
			Friend: "нет данных",
		}
		for _, p := range peers {
			matches, err := fetchMatchesWith(accountID, p.AccountID, limit)
			if err != nil {
				return "", err
			}
			winrate, games := calcWinrateFromMatches(matches)
			if games == 0 {
				continue
			}
			if winrate > best.Winrate || (winrate == best.Winrate && games > best.Games) {
				name := strings.TrimSpace(p.Personaname)
				if name == "" {
					name = fmt.Sprintf("Account %d", p.AccountID)
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

func runTelegramBot(token string, accountIDs []int64, heroes map[int]string) error {
	apiBase := fmt.Sprintf(telegramBaseURL, token)
	offset := 0
	for {
		url := fmt.Sprintf("%s/getUpdates?timeout=30&offset=%d", apiBase, offset)
		var resp telegramUpdatesResponse
		if err := getJSON(url, &resp); err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		if !resp.OK {
			return fmt.Errorf("telegram getUpdates failed: %s", resp.Description)
		}
		for _, upd := range resp.Result {
			offset = upd.UpdateID + 1
			if upd.Message == nil {
				continue
			}
			text := strings.TrimSpace(upd.Message.Text)
			if isStatCommand(text) {
				for _, accountID := range accountIDs {
					player, err := fetchPlayerProfile(accountID)
					if err != nil {
						sendTelegramMessage(apiBase, upd.Message.Chat.ID, fmt.Sprintf("Ошибка: %s", err.Error()), "")
						continue
					}
					matches, err := fetchRecentMatches(accountID)
					if err != nil {
						sendTelegramMessage(apiBase, upd.Message.Chat.ID, fmt.Sprintf("Ошибка: %s", err.Error()), "")
						continue
					}
					winrate := calcWinrate(matches, 20)
					if len(matches) > 10 {
						matches = matches[:10]
					}
					table := buildPlayerTable(matches, heroes, player.PersonaName)
					name := player.PersonaName
					if name == "" {
						name = "неизвестный"
					}
					header := fmt.Sprintf("<b>Последние матчи (%s)</b>\n<b>Winrate (за 20 игр): %.1f%%</b>\n<b>✅ победа, ❌ поражение</b>\n", escapeHTML(name), winrate)
					if player.AvatarFull != "" {
						if err := sendTelegramPhoto(apiBase, upd.Message.Chat.ID, player.AvatarFull, header, "HTML"); err != nil {
							return err
						}
						for _, msg := range buildTelegramMessages(table, "") {
							if err := sendTelegramMessage(apiBase, upd.Message.Chat.ID, msg, "HTML"); err != nil {
								return err
							}
						}
					} else {
						for _, msg := range buildTelegramMessages(table, header) {
							if err := sendTelegramMessage(apiBase, upd.Message.Chat.ID, msg, "HTML"); err != nil {
								return err
							}
						}
					}
				}
				continue
			}
			if isRatingCommand(text) {
				table, err := buildRatingTable(accountIDs)
				if err != nil {
					sendTelegramMessage(apiBase, upd.Message.Chat.ID, fmt.Sprintf("Ошибка: %s", err.Error()), "")
					continue
				}
				header := "<b>Рейтинг по Winrate (20)</b>\n"
				for _, msg := range buildTelegramMessages(table, header) {
					if err := sendTelegramMessage(apiBase, upd.Message.Chat.ID, msg, "HTML"); err != nil {
						return err
					}
				}
				continue
			}
			if ok, limit, err := parseFriendsCommand(text); ok {
				if err != nil {
					sendTelegramMessage(apiBase, upd.Message.Chat.ID, fmt.Sprintf("Ошибка: %s", err.Error()), "")
					continue
				}
				table, err := buildBestFriendsTable(accountIDs, limit)
				if err != nil {
					sendTelegramMessage(apiBase, upd.Message.Chat.ID, fmt.Sprintf("Ошибка: %s", err.Error()), "")
					continue
				}
				header := fmt.Sprintf("<b>Лучшие напарники по Winrate (за %d игр)</b>\n", limit)
				for _, msg := range buildTelegramMessages(table, header) {
					if err := sendTelegramMessage(apiBase, upd.Message.Chat.ID, msg, "HTML"); err != nil {
						return err
					}
				}
				continue
			}
			if isChatIDCommand(text) {
				if err := sendTelegramMessage(apiBase, upd.Message.Chat.ID, fmt.Sprintf("chat_id: %d", upd.Message.Chat.ID), ""); err != nil {
					return err
				}
				continue
			}
		}
	}
}

func isStatCommand(text string) bool {
	if text == "" {
		return false
	}
	if text == "/stat" {
		return true
	}
	if strings.HasPrefix(text, "/stat@") {
		return true
	}
	if strings.HasPrefix(text, "/stat ") {
		return true
	}
	return false
}

func isRatingCommand(text string) bool {
	if text == "" {
		return false
	}
	if text == "/rating" {
		return true
	}
	if strings.HasPrefix(text, "/rating@") {
		return true
	}
	if strings.HasPrefix(text, "/rating ") {
		return true
	}
	return false
}

func parseFriendsCommand(text string) (bool, int, error) {
	if text == "" {
		return false, 0, nil
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return false, 0, nil
	}
	cmd := strings.ToLower(fields[0])
	if strings.Contains(cmd, "@") {
		cmd = strings.SplitN(cmd, "@", 2)[0]
	}
	if cmd != "/friends" {
		return false, 0, nil
	}
	limit := 20
	if len(fields) > 1 {
		value, err := strconv.Atoi(fields[1])
		if err != nil || value <= 0 {
			return true, 0, fmt.Errorf("используй /friends <число>")
		}
		limit = value
	}
	return true, limit, nil
}

func isChatIDCommand(text string) bool {
	if text == "" {
		return false
	}
	if text == "/chatid" {
		return true
	}
	if strings.HasPrefix(text, "/chatid@") {
		return true
	}
	if strings.HasPrefix(text, "/chatid ") {
		return true
	}
	return false
}

func sendTelegramMessage(apiBase string, chatID int64, text string, parseMode string) error {
	payload := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}
	if parseMode != "" {
		payload["parse_mode"] = parseMode
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal telegram message: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, apiBase+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("telegram send failed: %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	return nil
}

func sendTelegramPhoto(apiBase string, chatID int64, photoURL string, caption string, parseMode string) error {
	payload := map[string]any{
		"chat_id": chatID,
		"photo":   photoURL,
	}
	if caption != "" {
		payload["caption"] = caption
	}
	if parseMode != "" {
		payload["parse_mode"] = parseMode
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal telegram photo: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, apiBase+"/sendPhoto", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram photo request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram photo send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("telegram photo failed: %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	return nil
}

func splitText(text string, maxLen int) []string {
	runes := []rune(text)
	if len(runes) <= maxLen {
		return []string{text}
	}
	var parts []string
	for len(runes) > maxLen {
		parts = append(parts, string(runes[:maxLen]))
		runes = runes[maxLen:]
	}
	if len(runes) > 0 {
		parts = append(parts, string(runes))
	}
	return parts
}

func escapeHTML(text string) string {
	var builder strings.Builder
	builder.Grow(len(text))
	for _, r := range text {
		switch r {
		case '&':
			builder.WriteString("&amp;")
		case '<':
			builder.WriteString("&lt;")
		case '>':
			builder.WriteString("&gt;")
		default:
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func buildTelegramMessages(report string, header string) []string {
	escaped := escapeHTML(report)

	preWrapperLen := runeLen("<pre></pre>")
	firstMax := telegramMaxLen - runeLen(header) - preWrapperLen
	if firstMax < 100 {
		firstMax = 100
	}
	if runeLen(escaped) == 0 {
		return []string{header + "<pre></pre>"}
	}

	firstParts := splitText(escaped, firstMax)
	first := firstParts[0]
	escapedRunes := []rune(escaped)
	firstLen := runeLen(first)
	escaped = string(escapedRunes[firstLen:])

	messages := []string{header + "<pre>" + first + "</pre>"}
	for runeLen(escaped) > 0 {
		parts := splitText(escaped, telegramMaxLen-preWrapperLen)
		part := parts[0]
		messages = append(messages, "<pre>"+part+"</pre>")
		partLen := runeLen(part)
		escapedRunes = []rune(escaped)
		escaped = string(escapedRunes[partLen:])
	}
	return messages
}

func runeLen(value string) int {
	return len([]rune(value))
}

func matchWin(m recentMatch) bool {
	return isWin(m.RadiantWin, m.PlayerSlot)
}

func calcWinrate(matches []recentMatch, limit int) float64 {
	winrate, _ := calcWinrateWithCount(matches, limit)
	return winrate
}

func calcWinrateWithCount(matches []recentMatch, limit int) (float64, int) {
	if limit <= 0 {
		return 0, 0
	}
	total := 0
	wins := 0
	for _, m := range matches {
		if total >= limit {
			break
		}
		total++
		if matchWin(m) {
			wins++
		}
	}
	if total == 0 {
		return 0, 0
	}
	return float64(wins) * 100 / float64(total), total
}

func calcWinrateFromMatches(matches []playerMatch) (float64, int) {
	if len(matches) == 0 {
		return 0, 0
	}
	wins := 0
	for _, m := range matches {
		if isWin(m.RadiantWin, m.PlayerSlot) {
			wins++
		}
	}
	return float64(wins) * 100 / float64(len(matches)), len(matches)
}

func isWin(radiantWin bool, playerSlot int) bool {
	isRadiant := playerSlot < 128
	return (radiantWin && isRadiant) || (!radiantWin && !isRadiant)
}

func formatDuration(seconds int) string {
	if seconds <= 0 {
		return "0:00"
	}
	min := seconds / 60
	sec := seconds % 60
	return fmt.Sprintf("%d:%02d", min, sec)
}

func trimTo(value string, max int) string {
	if len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

func monitorMatches(accountIDs []int64, heroes map[int]string, notify func(string)) {
	lastMatch := make(map[int64]int64, len(accountIDs))
	names := make(map[int64]string, len(accountIDs))
	if notify == nil {
		notify = func(msg string) {
			fmt.Println(msg)
		}
	}
	for _, accountID := range accountIDs {
		player, err := fetchPlayerProfile(accountID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "profile error: %s\n", err.Error())
		} else {
			names[accountID] = fallbackName(player.PersonaName)
		}
		matches, err := fetchRecentMatches(accountID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "matches error: %s\n", err.Error())
			continue
		}
		if len(matches) > 0 {
			lastMatch[accountID] = matches[0].MatchID
		}
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		for _, accountID := range accountIDs {
			matches, err := fetchRecentMatches(accountID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "matches error: %s\n", err.Error())
				continue
			}
			if len(matches) == 0 {
				continue
			}
			prev, ok := lastMatch[accountID]
			if !ok {
				lastMatch[accountID] = matches[0].MatchID
				continue
			}
			if matches[0].MatchID == prev {
				continue
			}
			var newMatches []recentMatch
			for _, m := range matches {
				if m.MatchID == prev {
					break
				}
				newMatches = append(newMatches, m)
			}
			for i := len(newMatches) - 1; i >= 0; i-- {
				line := formatMatchSummary(names[accountID], newMatches[i], heroes)
				notify(line)
			}
			lastMatch[accountID] = matches[0].MatchID
		}
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
	start := time.Unix(match.StartTime, 0).Local().Format("2006-01-02 15:04")
	duration := formatDuration(match.Duration)
	kda := fmt.Sprintf("%d/%d/%d", match.Kills, match.Deaths, match.Assists)
	return fmt.Sprintf("Новый матч: %s | %s | %s | %s | %s | %s", fallbackName(playerName), heroName, result, kda, duration, start)
}

func fallbackName(name string) string {
	if strings.TrimSpace(name) == "" {
		return "неизвестный"
	}
	return strings.TrimSpace(name)
}

func telegramNotifier(token string) func(string) {
	chatIDRaw := strings.TrimSpace(os.Getenv(telegramChatEnv))
	if chatIDRaw == "" {
		return nil
	}
	chatID, err := strconv.ParseInt(chatIDRaw, 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid %s: %s\n", telegramChatEnv, err.Error())
		return nil
	}
	apiBase := fmt.Sprintf(telegramBaseURL, token)
	return func(msg string) {
		if err := sendTelegramMessage(apiBase, chatID, msg, ""); err != nil {
			fmt.Fprintf(os.Stderr, "telegram notify error: %s\n", err.Error())
		}
	}
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
