package app

import (
	"fmt"
	"strconv"
	"strings"
)

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
