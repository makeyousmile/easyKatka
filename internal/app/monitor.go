package app

import (
	"fmt"
	"os"
	"time"
)

func monitorMatches(accountStore *accountIDStore, heroes map[int]string, notify func(matchNotification)) {
	accountIDs := accountStore.Get()
	lastMatch := make(map[int64]int64, len(accountIDs))
	names := make(map[int64]string, len(accountIDs))
	if notify == nil {
		notify = func(msg matchNotification) {
			fmt.Println(msg.Text)
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
		accountIDs = accountStore.Get()
		for _, accountID := range accountIDs {
			if _, ok := names[accountID]; !ok {
				player, err := fetchPlayerProfile(accountID)
				if err != nil {
					fmt.Fprintf(os.Stderr, "profile error: %s\n", err.Error())
				} else {
					names[accountID] = fallbackName(player.PersonaName)
				}
			}
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
				notify(matchNotification{
					Text:      formatMatchSummary(names[accountID], newMatches[i], heroes),
					MatchID:   newMatches[i].MatchID,
					AccountID: accountID,
				})
			}
			lastMatch[accountID] = matches[0].MatchID
		}
	}
}
