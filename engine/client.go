package engine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const endpoint = "wss://workers.gateway.onech.at"

func newSharedTransport() *http.Transport {
	return &http.Transport{
		MaxIdleConns:          25,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		DisableKeepAlives:     false,
		MaxConnsPerHost:       10,
		ResponseHeaderTimeout: 5 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{}
			conn, err := dialer.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			tcpConn, ok := conn.(*net.TCPConn)
			if ok {
				tcpConn.SetNoDelay(true)
			}
			return conn, nil
		},
	}
}

func sendReply(channelID, content, token, cfCookie string, client *http.Client) error {
	url := fmt.Sprintf("https://workers.api.onech.at/channels/%s/messages", channelID)
	body := []byte(fmt.Sprintf(`{"content":"%s"}`, content))

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))

	req.Header.Set("x-session-token", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://workers.onech.at")
	req.Header.Set("Referer", "https://workers.onech.at/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	if cfCookie != "" {
		req.Header.Set("Cookie", cfCookie)
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("status=%d body=%s", res.StatusCode, string(body))
	}
	return nil
}

func sendAllMessages(channelID string, messages []string, logFn func(string, ...interface{})) {
	for i, msg := range messages {
		if msg == "" {
			continue
		}
		if i > 0 {
			time.Sleep(10 * time.Millisecond)
		}
		logFn("Sending: %s", msg)
	}
}
