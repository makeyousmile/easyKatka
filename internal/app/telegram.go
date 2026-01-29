package app

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

func runTelegramBot(token string, accountIDs []int64, heroes map[int]string) error {
	apiBase := fmt.Sprintf(telegramBaseURL, token)
	offset := 0
	for {
		url := fmt.Sprintf("%s/getUpdates?timeout=30&offset=%d", apiBase, offset)
		var resp telegramUpdatesResponse
		if err := getJSON(url, &resp, nil); err != nil {
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
				header := "<b>Рейтинг по Winrate (50)</b>\n"
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
				header := fmt.Sprintf("<b>Лучшие напарники по Winrate (за последние %d игр)</b>\n", limit)
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
