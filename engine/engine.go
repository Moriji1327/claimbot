package engine

import (
	"io"
	"net/http"
	"sync"
	"time"
)

type Engine struct {
	config        *Config
	log           *LogEmitter
	bots          []*Bot
	mu            sync.RWMutex
	running       bool
	claimMu       sync.Mutex
	claimCount    int
	startTime     time.Time
	httpTransport *http.Transport
}

func NewEngine(cfg *Config, log *LogEmitter) *Engine {
	return &Engine{
		config:        cfg,
		log:           log,
		bots:          []*Bot{},
		claimCount:    0,
		httpTransport: newSharedTransport(),
	}
}

func (eng *Engine) StartAll() {
	eng.mu.Lock()
	defer eng.mu.Unlock()

	if eng.running {
		return
	}
	eng.running = true
	eng.startTime = time.Now()
	eng.claimCount = 0

	eng.prewarmHTTP()

	for i, token := range eng.config.SessionTokens {
		bot := NewBot(i+1, token, eng, eng.log)
		eng.bots = append(eng.bots, bot)
		bot.start()
		time.Sleep(500 * time.Millisecond)
	}

	eng.log.Infof(0, "Started %d bots", len(eng.config.SessionTokens))
}

func (eng *Engine) StopAll() {
	eng.mu.Lock()
	defer eng.mu.Unlock()

	if !eng.running {
		return
	}

	for _, bot := range eng.bots {
		bot.stop()
	}
	eng.bots = nil
	eng.running = false
	eng.log.Infof(0, "All bots stopped")
}

func (eng *Engine) RestartAll() {
	eng.log.Infof(0, "Restarting all bots...")
	eng.StopAll()
	time.Sleep(1 * time.Second)
	eng.StartAll()
}

func (eng *Engine) StopBot(id int) {
	eng.mu.Lock()
	defer eng.mu.Unlock()

	for _, bot := range eng.bots {
		if bot.id == id {
			bot.stop()
			eng.log.Infof(0, "Bot %d stopped", id)
			return
		}
	}
}

func (eng *Engine) StartBot(id int) {
	eng.mu.Lock()
	defer eng.mu.Unlock()

	for _, bot := range eng.bots {
		if bot.id == id {
			eng.log.Infof(0, "Bot %d already exists", id)
			return
		}
	}

	eng.log.Infof(0, "Starting bot %d...", id)
}

func (eng *Engine) Status() AppState {
	eng.mu.RLock()
	defer eng.mu.RUnlock()

	state := AppState{
		TotalClaims:  eng.claimCount,
		ServerCount:  len(eng.config.Servers),
		ChannelCount: len(channelCache),
	}

	if eng.running {
		uptime := time.Since(eng.startTime).Round(time.Second).String()
		for _, bot := range eng.bots {
			botState := BotState{
				ID:     bot.id,
				Status: bot.Status(),
				Uptime: uptime,
			}
			state.Bots = append(state.Bots, botState)
		}
	}

	return state
}

func (eng *Engine) IsRunning() bool {
	eng.mu.RLock()
	defer eng.mu.RUnlock()
	return eng.running
}

func (eng *Engine) GetConfig() *Config {
	return eng.config
}

func (eng *Engine) UpdateConfig(cfg *Config) {
	eng.mu.Lock()
	eng.config = cfg
	eng.mu.Unlock()
}

func (eng *Engine) prewarmHTTP() {
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: eng.httpTransport,
	}
	req, _ := http.NewRequest("GET", "https://workers.api.onech.at/", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	eng.log.Infof(0, "HTTP connection pre-warmed")
}
