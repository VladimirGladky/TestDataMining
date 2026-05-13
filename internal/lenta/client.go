package lenta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"time"

	"TestDataMining/internal/config"
)

const (
	baseURL           = "https://lenta.com"
	selectionsPath    = "/api-gateway/v1/catalog/items/selections"
	productURLPattern = "https://lenta.com/product/%s-%d/"
)

type Client struct {
	httpClient *http.Client
	cfg        *config.Config
	retries    int
}

func NewClient(cfg *config.Config) (*Client, error) {
	proxyURL, err := url.Parse(cfg.ProxyURL)
	if err != nil {
		return nil, fmt.Errorf("parse proxy url: %w", err)
	}

	transport := &http.Transport{
		Proxy:               http.ProxyURL(proxyURL),
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 15 * time.Second,
		DisableCompression:  false,
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   time.Duration(cfg.RequestTimeout) * time.Second,
		},
		cfg:     cfg,
		retries: cfg.RetryAttempts,
	}, nil
}

func (c *Client) FetchSelectionPage(ctx context.Context, selectionID int64, offset, limit int) (*SelectionResponse, error) {
	body := SelectionRequest{
		SelectionID:      selectionID,
		SelectionGroupID: nil,
		Filters:          Filters{Checkbox: []any{}, Multicheckbox: []any{}, Range: []any{}},
		Sort:             Sort{Type: "popular", Order: "desc"},
		Limit:            limit,
		Offset:           offset,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			delay := backoff(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := c.doSelections(ctx, payload)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("after %d attempts: %w", c.retries+1, lastErr)
}

func (c *Client) doSelections(ctx context.Context, payload []byte) (*SelectionResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+selectionsPath, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	c.applyHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 50*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		preview := string(bodyBytes)
		if len(preview) > 300 {
			preview = preview[:300]
		}
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, preview)
	}

	var out SelectionResponse
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &out, nil
}

func (c *Client) applyHeaders(req *http.Request) {
	traceID := randHex(16)
	spanID := randHex(8)

	h := req.Header
	h.Set("accept", "application/json")
	h.Set("accept-language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
	h.Set("client", "angular_web_0.0.2")
	h.Set("content-type", "application/json")
	h.Set("origin", baseURL)
	h.Set("referer", baseURL+"/")
	h.Set("priority", "u=1, i")
	h.Set("sec-ch-ua", `"Google Chrome";v="147", "Not.A/Brand";v="8", "Chromium";v="147"`)
	h.Set("sec-ch-ua-mobile", "?0")
	h.Set("sec-ch-ua-platform", `"macOS"`)
	h.Set("sec-fetch-dest", "empty")
	h.Set("sec-fetch-mode", "cors")
	h.Set("sec-fetch-site", "same-origin")
	h.Set("user-agent", c.cfg.UserAgent)
	h.Set("cookie", c.cfg.Cookie)
	h.Set("deviceid", c.cfg.DeviceID)
	h.Set("sessiontoken", c.cfg.SessionToken)
	h.Set("x-device-id", c.cfg.DeviceID)
	h.Set("x-user-session-id", c.cfg.UserSessionID)
	h.Set("x-delivery-mode", c.cfg.DeliveryMode)
	h.Set("x-domain", c.cfg.Domain)
	h.Set("x-device-os", "Web")
	h.Set("x-device-os-version", "12.4.8")
	h.Set("x-device-web-platform", "desktop_web")
	h.Set("x-platform", "omniweb")
	h.Set("x-retail-brand", "lo")
	h.Set("x-trace-id", traceID)
	h.Set("x-span-id", spanID)
	h.Set("traceparent", fmt.Sprintf("00-%s-%s-01", traceID, spanID))
}

func ProductURL(slug string, id int64) string {
	return fmt.Sprintf(productURLPattern, slug, id)
}

func backoff(attempt int) time.Duration {
	base := time.Duration(500*attempt*attempt) * time.Millisecond
	jitter := time.Duration(rand.Int64N(500)) * time.Millisecond
	return base + jitter
}

func randHex(n int) string {
	const hex = "0123456789abcdef"
	b := make([]byte, n*2)
	for i := range b {
		b[i] = hex[rand.IntN(16)]
	}
	return string(b)
}
