package engine

import (
	"regexp"
	"sync"
)

type ServerConfig struct {
	Name                string   `json:"name"`
	Messages            []string `json:"messages"`
	UseTicketNumber     bool     `json:"useTicketNumber"`
	TicketPrefix        string   `json:"ticketPrefix,omitempty"`
	CategoryNamePattern string   `json:"categoryNamePattern"`
	UnclaimReply        string   `json:"unclaimReply"`
	RaffleReply         string   `json:"raffleReply"`
	AggressiveMode      bool     `json:"aggressiveMode"`
	Disabled            bool     `json:"disabled,omitempty"`
	TriggerClaim        string   `json:"triggerClaim"`
	TriggerUnclaim      string   `json:"triggerUnclaim"`
	TriggerReopened     string   `json:"triggerReopened"`
	TriggerRaffle       string   `json:"triggerRaffle"`
}

type ChanInfo struct {
	ServerID   string
	Name       string
	CategoryID string
}

type CategoryInfo struct {
	ID   string
	Name string
}

type BotStatus string

const (
	StatusDisconnected BotStatus = "disconnected"
	StatusConnecting   BotStatus = "connecting"
	StatusConnected    BotStatus = "connected"
	StatusError        BotStatus = "error"
)

type LogLevel string

const (
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

type LogEntry struct {
	Timestamp string   `json:"timestamp"`
	Level     LogLevel `json:"level"`
	BotID     int      `json:"botID"`
	Message   string   `json:"message"`
}

type BotState struct {
	ID        int       `json:"id"`
	Status    BotStatus `json:"status"`
	Uptime    string    `json:"uptime"`
	Claims    int       `json:"claims"`
	LastClaim string    `json:"lastClaim"`
}

type AppState struct {
	Bots         []BotState `json:"bots"`
	TotalClaims  int        `json:"totalClaims"`
	ServerCount  int        `json:"serverCount"`
	ChannelCount int        `json:"channelCount"`
}

type Config struct {
	SessionTokens   []string                `json:"sessionTokens"`
	Servers         map[string]ServerConfig `json:"servers"`
	Port            int                     `json:"port"`
	AutoStart       bool                    `json:"autoStart"`
	DefaultTriggers DefaultTriggers         `json:"defaultTriggers"`
	CfCookie        string                  `json:"cfCookie"`
}

type DefaultTriggers struct {
	Claim    string `json:"claim"`
	Unclaim  string `json:"unclaim"`
	Reopened string `json:"reopened"`
	Raffle   string `json:"raffle"`
}

var (
	NumRegex = regexp.MustCompile(`\d+`)

	channelCache = make(map[string]ChanInfo)
	cacheMu      sync.RWMutex

	categoryCache = make(map[string]CategoryInfo)
	categoryMu    sync.RWMutex

	serverCategories = make(map[string][]string)
	categoriesMu     sync.RWMutex

	claimed   = make(map[string]bool)
	claimedMu sync.RWMutex
)

func ClaimedSet(key string, val bool) {
	claimedMu.Lock()
	claimed[key] = val
	claimedMu.Unlock()
}

func GetClaimed() map[string]bool {
	claimedMu.RLock()
	defer claimedMu.RUnlock()
	m := make(map[string]bool, len(claimed))
	for k, v := range claimed {
		m[k] = v
	}
	return m
}

type replyJob struct {
	channelID string
	content   string
}

const replyChanSize = 100
const replyWorkerCount = 3

type messageEvent struct {
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Content string `json:"content"`
}

func FillTriggers(cfg *ServerConfig, def DefaultTriggers) {
	if cfg.TriggerClaim == "" {
		cfg.TriggerClaim = def.Claim
	}
	if cfg.TriggerUnclaim == "" {
		cfg.TriggerUnclaim = def.Unclaim
	}
	if cfg.TriggerReopened == "" {
		cfg.TriggerReopened = def.Reopened
	}
	if cfg.TriggerRaffle == "" {
		cfg.TriggerRaffle = def.Raffle
	}
}
