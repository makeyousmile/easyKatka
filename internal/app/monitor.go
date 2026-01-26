package app

import (
	"fmt"
	"os"
	"time"
)

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
