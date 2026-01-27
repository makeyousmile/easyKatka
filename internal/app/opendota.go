package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
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

type rateLimiter struct {
	tick <-chan time.Time
}

func newRateLimiter(max int, per time.Duration) *rateLimiter {
	if max <= 0 {
		return nil
	}
	interval := per / time.Duration(max)
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	return &rateLimiter{tick: ticker.C}
}

func (l *rateLimiter) Wait() {
	if l == nil {
		return
	}
	<-l.tick
}

func fetchHeroes() (map[int]string, error) {
	var heroes []hero
	if err := getOpendotaJSON(baseURL+heroesURL, &heroes); err != nil {
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
	if err := getOpendotaJSON(url, &matches); err != nil {
		return nil, err
	}
	return matches, nil
}

func fetchPlayerProfile(accountID int64) (playerProfileData, error) {
	var player playerProfile
	url := fmt.Sprintf(baseURL+playerURL, accountID)
	if err := getOpendotaJSON(url, &player); err != nil {
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
	if err := getOpendotaJSON(url, &peers); err != nil {
		return nil, err
	}
	return peers, nil
}

func fetchMatchesWith(accountID int64, includedAccountID int64, limit int) ([]playerMatch, error) {
	var matches []playerMatch
	url := fmt.Sprintf("%s"+playerMatchesURL+"?included_account_id=%d&limit=%d", baseURL, accountID, includedAccountID, limit)
	if err := getOpendotaJSON(url, &matches); err != nil {
		return nil, err
	}
	return matches, nil
}

func getOpendotaJSON(url string, out any) error {
	return getJSON(url, out, opendotaLimiter)
}

func getJSON(url string, out any, limiter *rateLimiter) error {
	if limiter != nil {
		limiter.Wait()
	}
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
