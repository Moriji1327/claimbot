package engine

import (
	"bytes"
	"strings"
)

func quickExtractField(raw []byte, field string) string {
	prefix := []byte(`"` + field + `":"`)
	start := bytes.Index(raw, prefix)
	if start == -1 {
		return ""
	}
	start += len(prefix)
	end := bytes.IndexByte(raw[start:], '"')
	if end == -1 {
		return ""
	}
	return string(raw[start : start+end])
}

func isMessageEvent(raw []byte) bool {
	return bytes.Contains(raw, []byte(`"type":"Message"`))
}

func containsTrigger(content string, cfg ServerConfig) bool {
	lower := strings.ToLower(content)
	return strings.Contains(lower, strings.ToLower(cfg.TriggerClaim)) ||
		strings.Contains(lower, strings.ToLower(cfg.TriggerUnclaim)) ||
		strings.Contains(lower, strings.ToLower(cfg.TriggerReopened)) ||
		strings.Contains(lower, strings.ToLower(cfg.TriggerRaffle))
}

func detectTriggerType(content string, cfg ServerConfig) string {
	lower := strings.ToLower(content)
	if strings.Contains(lower, strings.ToLower(cfg.TriggerRaffle)) {
		return "raffle"
	}
	if strings.Contains(lower, strings.ToLower(cfg.TriggerUnclaim)) ||
		strings.Contains(lower, strings.ToLower(cfg.TriggerReopened)) {
		return "unclaim"
	}
	if strings.Contains(lower, strings.ToLower(cfg.TriggerClaim)) {
		return "claim"
	}
	return ""
}

func extractTicketNumber(name string) string {
	if match := NumRegex.FindString(name); match != "" {
		return match
	}
	l := strings.ToLower(name)
	if strings.HasPrefix(l, "ticket-") || strings.HasPrefix(l, "ticket_") {
		return strings.TrimPrefix(strings.TrimPrefix(l, "ticket-"), "ticket_")
	}
	return ""
}

func shouldReplyToChannel(channelID, channelName string, cfg ServerConfig) bool {
	if cfg.CategoryNamePattern == "" {
		return true
	}

	cacheMu.RLock()
	info, exists := channelCache[channelID]
	cacheMu.RUnlock()

	if !exists {
		return strings.Contains(strings.ToLower(channelName), strings.ToLower(cfg.CategoryNamePattern))
	}

	patternLower := strings.ToLower(cfg.CategoryNamePattern)

	if info.CategoryID != "" {
		categoryMu.RLock()
		catInfo, catExists := categoryCache[info.CategoryID]
		categoryMu.RUnlock()

		if catExists && strings.Contains(strings.ToLower(catInfo.Name), patternLower) {
			return true
		}
	}

	return strings.Contains(strings.ToLower(channelName), patternLower)
}
