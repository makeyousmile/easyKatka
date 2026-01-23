package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

	telegramTokenEnv = "TELEGRAM_BOT_TOKEN"
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
	} `json:"profile"`
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

func fetchPlayerName(accountID int64) (string, error) {
	var player playerProfile
	url := fmt.Sprintf(baseURL+playerURL, accountID)
	if err := getJSON(url, &player); err != nil {
		return "", err
	}
	return strings.TrimSpace(player.Profile.PersonaName), nil
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
		playerName, err := fetchPlayerName(accountID)
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

		writeMatches(&builder, matches, heroes, playerName)
		if i < len(accountIDs)-1 {
			builder.WriteString("\n\n")
		}
	}
	return builder.String(), nil
}

func writeMatches(writer io.Writer, matches []recentMatch, heroes map[int]string, playerName string) {
	if playerName == "" {
		playerName = "неизвестный"
	}
	fmt.Fprintf(writer, "Последние матчи (%s):\n", playerName)
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
			if !isStatCommand(text) {
				continue
			}
			report, err := buildReport(accountIDs, heroes)
			if err != nil {
				sendTelegramMessage(apiBase, upd.Message.Chat.ID, fmt.Sprintf("Ошибка: %s", err.Error()), "")
				continue
			}
			for _, msg := range buildTelegramMessages(report) {
				if err := sendTelegramMessage(apiBase, upd.Message.Chat.ID, msg, "HTML"); err != nil {
					return err
				}
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

func buildTelegramMessages(report string) []string {
	header := "<b>Последние матчи</b>\n<b>W=победа ✅, L=поражение ❌</b>\n"
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
	isRadiant := m.PlayerSlot < 128
	return (m.RadiantWin && isRadiant) || (!m.RadiantWin && !isRadiant)
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

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
