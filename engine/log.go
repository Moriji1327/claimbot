package engine

import (
	"fmt"
	"sync"
	"time"
)

type subscriber struct {
	ch     chan LogEntry
	active bool
}

type LogEmitter struct {
	mu      sync.Mutex
	entries chan LogEntry
	subs    []*subscriber
}

func NewLogEmitter(buffer int) *LogEmitter {
	return &LogEmitter{
		entries: make(chan LogEntry, buffer),
		subs:    []*subscriber{},
	}
}

func (le *LogEmitter) Start() {
	go func() {
		for entry := range le.entries {
			le.mu.Lock()
			for _, sub := range le.subs {
				if !sub.active {
					continue
				}
				select {
				case sub.ch <- entry:
				default:
				}
			}
			le.mu.Unlock()
		}
	}()
}

func (le *LogEmitter) Subscribe() chan LogEntry {
	ch := make(chan LogEntry, 256)
	sub := &subscriber{ch: ch, active: true}
	le.mu.Lock()
	le.subs = append(le.subs, sub)
	le.mu.Unlock()
	return ch
}

func (le *LogEmitter) Unsubscribe(ch chan LogEntry) {
	le.mu.Lock()
	defer le.mu.Unlock()
	for _, sub := range le.subs {
		if sub.ch == ch {
			sub.active = false
		}
	}
}

func (le *LogEmitter) Emit(botID int, level LogLevel, msg string) {
	entry := LogEntry{
		Timestamp: time.Now().Format("15:04:05"),
		Level:     level,
		BotID:     botID,
		Message:   msg,
	}
	select {
	case le.entries <- entry:
	default:
	}
}

func (le *LogEmitter) Infof(botID int, format string, args ...interface{}) {
	le.Emit(botID, LevelInfo, fmt.Sprintf(format, args...))
}

func (le *LogEmitter) Warnf(botID int, format string, args ...interface{}) {
	le.Emit(botID, LevelWarn, fmt.Sprintf(format, args...))
}

func (le *LogEmitter) Errorf(botID int, format string, args ...interface{}) {
	le.Emit(botID, LevelError, fmt.Sprintf(format, args...))
}
