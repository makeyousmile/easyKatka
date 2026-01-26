package app

import "time"

const (
	baseURL          = "https://api.opendota.com/api"
	steamID64Offset  = int64(76561197960265728)
	maxUint32        = int64(^uint32(0))
	requestTimeout   = 15 * time.Second
	opendotaRateCap  = 60
	opendotaRateSpan = time.Minute
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

var opendotaLimiter = newRateLimiter(opendotaRateCap, opendotaRateSpan)
