package app

import (
	"fmt"
	"strings"
)

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

func fallbackName(name string) string {
	if strings.TrimSpace(name) == "" {
		return "неизвестный"
	}
	return strings.TrimSpace(name)
}
