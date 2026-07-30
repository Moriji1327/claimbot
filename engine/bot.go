package engine

import (
	"context"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Bot struct {
	id         int
	token      string
	engine     *Engine
	httpClient *http.Client
	log        *LogEmitter
	ctx        context.Context
	cancel     context.CancelFunc
	status     BotStatus
	replies    chan replyJob
}

func NewBot(id int, token string, eng *Engine, log *LogEmitter) *Bot {
	return &Bot{
		id:         id,
		token:      token,
		engine:     eng,
		httpClient: &http.Client{
			Timeout:   5 * time.Second,
			Transport: eng.httpTransport,
		},
		log:     log,
		status:  StatusDisconnected,
		replies: make(chan replyJob, replyChanSize),
	}
}

func (b *Bot) start() {
	b.ctx, b.cancel = context.WithCancel(context.Background())
	b.status = StatusConnecting

	tokenDisplay := b.token
	if len(tokenDisplay) > 20 {
		tokenDisplay = tokenDisplay[:20] + "..."
	}
	b.log.Infof(b.id, "Starting with token: %s", tokenDisplay)

	go b.run()
	for i := 0; i < replyWorkerCount; i++ {
		go b.replyWorker()
	}
}

func (b *Bot) stop() {
	if b.cancel != nil {
		b.cancel()
	}
	b.status = StatusDisconnected
	b.log.Infof(b.id, "Stopped")
}

func (b *Bot) replyWorker() {
	for {
		select {
		case <-b.ctx.Done():
			return
		case job, ok := <-b.replies:
			if !ok {
				return
			}
			err := sendReply(job.channelID, job.content, b.token, b.engine.config.CfCookie, b.httpClient)
			if err != nil {
				b.log.Warnf(b.id, "Failed: %s - %v", job.content, err)
			} else {
				b.log.Infof(b.id, "Sent: %s", job.content)
			}
		}
	}
}

func (b *Bot) run() {
	for {
		select {
		case <-b.ctx.Done():
			return
		default:
		}

		wsURL := endpoint + "?token=" + b.token

		headers := http.Header{}
		headers.Set("Origin", "https://workers.onech.at")
		headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		headers.Set("Accept-Language", "en-US,en;q=0.9")
		headers.Set("x-session-token", b.token)
		if cookie := b.engine.config.CfCookie; cookie != "" {
			headers.Set("Cookie", cookie)
		}

		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, headers)
		if err != nil {
			b.log.Warnf(b.id, "WS error: %v", err)
			if resp != nil {
				b.log.Warnf(b.id, "Status: %d", resp.StatusCode)
				if resp.Body != nil {
					body, _ := io.ReadAll(resp.Body)
					b.log.Warnf(b.id, "Response: %s", string(body))
					resp.Body.Close()
				}
			}
			b.status = StatusError
			delay := time.Duration(2+rand.Intn(4)) * time.Second
			select {
			case <-b.ctx.Done():
				return
			case <-time.After(delay):
			}
			continue
		}
		b.log.Infof(b.id, "Connected")
		b.status = StatusConnected

		auth := map[string]string{"type": "Authenticate", "token": b.token}
		if err := conn.WriteJSON(auth); err != nil {
			b.log.Warnf(b.id, "Auth error: %v", err)
			conn.Close()
			b.status = StatusError
			delay := time.Duration(2+rand.Intn(4)) * time.Second
			select {
			case <-b.ctx.Done():
				return
			case <-time.After(delay):
			}
			continue
		}
		b.log.Infof(b.id, "Authenticated")

		b.readLoop(conn)
		conn.Close()
		b.status = StatusDisconnected

		delay := time.Duration(1+rand.Intn(3)) * time.Second
		select {
		case <-b.ctx.Done():
			return
		case <-time.After(delay):
		}
	}
}

func (b *Bot) Status() BotStatus {
	return b.status
}
