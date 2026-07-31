package engine

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

var blockedChannels = map[string]bool{}

func (b *Bot) handleMessage(raw []byte) {
	var evt messageEvent
	if err := json.Unmarshal(raw, &evt); err != nil {
		return
	}

	if evt.Channel == "" || evt.Content == "" || blockedChannels[evt.Channel] {
		return
	}

	cfg, ok := b.engine.GetServerConfig(evt.Channel)
	if !ok {
		return
	}

	if cfg.Disabled {
		return
	}

	if !containsTrigger(evt.Content, cfg) {
		return
	}

	triggerType := detectTriggerType(evt.Content, cfg)
	if triggerType == "" {
		return
	}

	var replies []string
	switch triggerType {
	case "claim":
		claimedMu.RLock()
		alreadyClaimed := claimed[evt.Channel]
		claimedMu.RUnlock()

		if alreadyClaimed {
			return
		}

		cacheMu.RLock()
		info, cached := channelCache[evt.Channel]
		cacheMu.RUnlock()
		if !cached {
			return
		}

		if !shouldReplyToChannel(evt.Channel, info.Name, cfg) {
			return
		}

		claimedMu.Lock()
		claimed[evt.Channel] = true
		claimedMu.Unlock()

		if cfg.UseTicketNumber {
			ticket := extractTicketNumber(info.Name)
			if ticket != "" {
				replies = []string{cfg.TicketPrefix + ticket}
			}
		} else {
			replies = cfg.Messages
		}

		b.engine.claimMu.Lock()
		b.engine.claimCount++
		b.engine.claimMu.Unlock()

	case "unclaim":
		if cfg.UnclaimReply != "" {
			replies = []string{cfg.UnclaimReply}
			claimedMu.Lock()
			delete(claimed, evt.Channel)
			claimedMu.Unlock()
		}

	case "raffle":
		if cfg.RaffleReply != "" {
			replies = []string{cfg.RaffleReply}
		}
	}

	if len(replies) == 0 {
		return
	}

	b.log.Infof(b.id, "⚡ %s trigger: %s", triggerType, evt.Channel)

	for _, msg := range replies {
		if msg == "" {
			continue
		}
		select {
		case b.replies <- replyJob{channelID: evt.Channel, content: msg}:
		default:
			b.log.Warnf(b.id, "Reply channel full, dropping: %s", msg)
		}
	}
}

func (b *Bot) handleEvent(evt map[string]interface{}) {
	typ, _ := evt["type"].(string)

	switch typ {
	case "Ready":
		if chs, ok := evt["channels"].([]interface{}); ok {
			cacheMu.Lock()
			for _, c := range chs {
				if cm, ok := c.(map[string]interface{}); ok {
					id, _ := cm["_id"].(string)
					srv, _ := cm["server"].(string)
					name, _ := cm["name"].(string)
					cat, _ := cm["category"].(string)
					if id != "" {
						channelCache[id] = ChanInfo{
							ServerID:   srv,
							Name:       name,
							CategoryID: cat,
						}
					}
				}
			}
			cacheMu.Unlock()
			b.log.Infof(b.id, "Cached %d channels", len(channelCache))
		}

		if servers, ok := evt["servers"].([]interface{}); ok {
			categoriesMu.Lock()
			categoryMu.Lock()
			for _, s := range servers {
				if sm, ok := s.(map[string]interface{}); ok {
					srvID, _ := sm["_id"].(string)
					if cats, ok := sm["categories"].([]interface{}); ok {
						var catIDs []string
						for _, cat := range cats {
							if catMap, ok := cat.(map[string]interface{}); ok {
								catID, _ := catMap["id"].(string)
								catName, _ := catMap["title"].(string)
								if catID != "" {
									catIDs = append(catIDs, catID)
									categoryCache[catID] = CategoryInfo{
										ID:   catID,
										Name: catName,
									}
								}
							}
						}
						if len(catIDs) > 0 {
							serverCategories[srvID] = catIDs
						}
					}
				}
			}
			categoryMu.Unlock()
			categoriesMu.Unlock()
		}

	case "ChannelCreate":
		id, _ := evt["_id"].(string)
		srv, _ := evt["server"].(string)
		name, _ := evt["name"].(string)
		cat := ""
		if catVal, ok := evt["category"].(string); ok {
			cat = catVal
		}

		if id == "" {
			return
		}

		cacheMu.Lock()
		channelCache[id] = ChanInfo{
			ServerID:   srv,
			Name:       name,
			CategoryID: cat,
		}
		cacheMu.Unlock()

		if srv == "" {
			return
		}

		serverCfg, ok := b.engine.GetServerConfig(id)
		if !ok || !serverCfg.AggressiveMode || serverCfg.Disabled {
			return
		}

		if !shouldReplyToChannel(id, name, serverCfg) {
			return
		}

		claimedMu.Lock()
		if claimed[id] {
			claimedMu.Unlock()
			return
		}
		claimed[id] = true
		claimedMu.Unlock()

		var replies []string
		if serverCfg.UseTicketNumber {
			ticket := extractTicketNumber(name)
			if ticket != "" {
				replies = []string{serverCfg.TicketPrefix + ticket}
			}
		} else {
			replies = serverCfg.Messages
		}

		if len(replies) > 0 {
			b.log.Infof(b.id, "AGGRESSIVE: %s", id)
			for _, msg := range replies {
				if msg == "" {
					continue
				}
				select {
				case b.replies <- replyJob{channelID: id, content: msg}:
				default:
					b.log.Warnf(b.id, "Reply channel full, dropping: %s", msg)
				}
			}
			b.engine.claimMu.Lock()
			b.engine.claimCount++
			b.engine.claimMu.Unlock()
		}
	}
}

func (b *Bot) readLoop(conn *websocket.Conn) {
	for {
		select {
		case <-b.ctx.Done():
			return
		default:
		}

		_, raw, err := conn.ReadMessage()
		if err != nil {
			b.log.Warnf(b.id, "Read error: %v", err)
			return
		}

		if isMessageEvent(raw) {
			b.handleMessage(raw)
		} else {
			var evt map[string]interface{}
			if json.Unmarshal(raw, &evt) == nil {
				b.handleEvent(evt)
			}
		}
	}
}

func (eng *Engine) GetServerConfig(channelID string) (ServerConfig, bool) {
	cacheMu.RLock()
	info, cached := channelCache[channelID]
	cacheMu.RUnlock()

	if !cached || info.ServerID == "" {
		return ServerConfig{}, false
	}

	cfg, ok := eng.config.Servers[info.ServerID]
	if !ok {
		return ServerConfig{}, false
	}

	return cfg, true
}

func init() {
	log.SetFlags(log.Ltime | log.Lshortfile)
}
