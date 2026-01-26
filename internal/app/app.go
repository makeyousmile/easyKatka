package app

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func Run() error {
	accountIDs, err := loadAccountIDs("account_id")
	if err != nil {
		return err
	}

	heroes, err := fetchHeroes()
	if err != nil {
		return err
	}

	telegramToken := strings.TrimSpace(os.Getenv(telegramTokenEnv))
	if telegramToken != "" {
		if notify := telegramNotifier(telegramToken); notify != nil {
			go monitorMatches(accountIDs, heroes, notify)
		}
		return runTelegramBot(telegramToken, accountIDs, heroes)
	}

	report, err := buildReport(accountIDs, heroes)
	if err != nil {
		return err
	}
	fmt.Print(report)

	monitorMatches(accountIDs, heroes, nil)
	return nil
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
