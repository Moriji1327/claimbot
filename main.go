package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"sentry/engine"
	"sentry/web"
)

var (
	configPath  string
	claimedPath string
	eng         *engine.Engine
	logEmitter  *engine.LogEmitter
	sseClients  = make(map[chan engine.LogEntry]bool)
)

const configFileName = "config.json"
const claimedFileName = "claimed.json"

func getExecDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func main() {
	dir := getExecDir()
	configPath = filepath.Join(dir, configFileName)
	claimedPath = filepath.Join(dir, claimedFileName)

	logEmitter = engine.NewLogEmitter(1024)
	logEmitter.Start()

	cfg, err := engine.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if port := os.Getenv("PORT"); port != "" {
		if p, err := fmt.Sscanf(port, "%d", &cfg.Port); err == nil && p == 1 {
			log.Printf("Using PORT from environment: %d", cfg.Port)
		}
	}

	claimedMap, err := engine.LoadClaimed(claimedPath)
	if err != nil {
		log.Printf("Warning: failed to load claimed channels: %v", err)
	}
	for k, v := range claimedMap {
		engine.ClaimedSet(k, v)
	}

	eng = engine.NewEngine(cfg, logEmitter)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/config", handleConfig)
	mux.HandleFunc("/api/status", handleStatus)
	mux.HandleFunc("/api/bot/start", handleBotStart)
	mux.HandleFunc("/api/bot/stop", handleBotStop)
	mux.HandleFunc("/api/bot/restart", handleBotRestart)
	mux.HandleFunc("/api/logs", handleLogs)
	mux.HandleFunc("/favicon.ico", handleFavicon)
	mux.HandleFunc("/", handleStatic)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}

	go func() {
		log.Printf("Dashboard: http://localhost:%d", cfg.Port)
		openBrowser(fmt.Sprintf("http://localhost:%d", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	if cfg.AutoStart {
		logEmitter.Infof(0, "Auto-starting bots...")
		eng.StartAll()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sigCh

	log.Println("Shutting down...")
	eng.StopAll()
	saveClaimed()
	os.Exit(0)
}

func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	exec.Command(cmd, args...).Start()
}

func saveClaimed() {
	claimedMap := engine.GetClaimed()
	if err := engine.SaveClaimed(claimedPath, claimedMap); err != nil {
		log.Printf("Failed to save claimed channels: %v", err)
	}
}

func handleFavicon(w http.ResponseWriter, r *http.Request) {
	data, err := web.Files.ReadFile("icon.ico")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/x-icon")
	w.Write(data)
}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		data, err := web.Files.ReadFile("index.html")
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
		return
	}

	filePath := strings.TrimPrefix(r.URL.Path, "/static/")
	if filePath == "" {
		http.NotFound(w, r)
		return
	}

	data, err := web.Files.ReadFile(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	contentType := "text/plain"
	if strings.HasSuffix(filePath, ".css") {
		contentType = "text/css; charset=utf-8"
	} else if strings.HasSuffix(filePath, ".js") {
		contentType = "application/javascript; charset=utf-8"
	} else if strings.HasSuffix(filePath, ".ico") {
		contentType = "image/x-icon"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write(data)
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		cfg := eng.GetConfig()
		resp := map[string]interface{}{
			"sessionTokens":   cfg.SessionTokens,
			"servers":         cfg.Servers,
			"port":            cfg.Port,
			"autoStart":       cfg.AutoStart,
			"defaultTriggers": cfg.DefaultTriggers,
		}
		writeJSON(w, resp)

	case "POST":
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var req map[string]interface{}
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		cfg := eng.GetConfig()

		if tokens, ok := req["_tokens"]; ok {
			tokensArr, _ := tokens.([]interface{})
			cfg.SessionTokens = make([]string, len(tokensArr))
			for i, t := range tokensArr {
				cfg.SessionTokens[i], _ = t.(string)
			}
		} else if defaults, ok := req["_defaults"]; ok {
			if defMap, ok := defaults.(map[string]interface{}); ok {
				if v, ok := defMap["claim"]; ok { cfg.DefaultTriggers.Claim, _ = v.(string) }
				if v, ok := defMap["unclaim"]; ok { cfg.DefaultTriggers.Unclaim, _ = v.(string) }
				if v, ok := defMap["reopened"]; ok { cfg.DefaultTriggers.Reopened, _ = v.(string) }
				if v, ok := defMap["raffle"]; ok { cfg.DefaultTriggers.Raffle, _ = v.(string) }
			}
		} else if port, ok := req["_port"]; ok {
			if p, ok := port.(float64); ok {
				cfg.Port = int(p)
			}
		} else if serverId, ok := req["serverId"]; ok {
			sid, _ := serverId.(string)

			if del, ok := req["_delete"]; ok && del.(bool) {
				delete(cfg.Servers, sid)
			} else {
				srv := engine.ServerConfig{
					Name:                getString(req, "name"),
					CategoryNamePattern: getString(req, "categoryNamePattern"),
					Messages:            getStringSlice(req, "messages"),
					UnclaimReply:        getString(req, "unclaimReply"),
					RaffleReply:         getString(req, "raffleReply"),
					UseTicketNumber:     getBool(req, "useTicketNumber"),
					AggressiveMode:      getBool(req, "aggressiveMode"),
					TriggerClaim:        getString(req, "triggerClaim"),
					TriggerUnclaim:      getString(req, "triggerUnclaim"),
					TriggerReopened:     getString(req, "triggerReopened"),
					TriggerRaffle:       getString(req, "triggerRaffle"),
				}
				engine.FillTriggers(&srv, cfg.DefaultTriggers)
				cfg.Servers[sid] = srv
			}
		}

		eng.UpdateConfig(cfg)
		if err := engine.SaveConfig(configPath, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	state := eng.Status()
	writeJSON(w, state)
}

func handleBotStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	eng.StartAll()
	writeJSON(w, map[string]string{"status": "started"})
}

func handleBotStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	eng.StopAll()
	saveClaimed()
	writeJSON(w, map[string]string{"status": "stopped"})
}

func handleBotRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	eng.RestartAll()
	writeJSON(w, map[string]string{"status": "restarted"})
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	ch := logEmitter.Subscribe()
	defer logEmitter.Unsubscribe(ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(entry)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getStringSlice(m map[string]interface{}, key string) []string {
	if v, ok := m[key]; ok {
		if arr, ok := v.([]interface{}); ok {
			result := make([]string, len(arr))
			for i, item := range arr {
				result[i], _ = item.(string)
			}
			return result
		}
	}
	return nil
}
