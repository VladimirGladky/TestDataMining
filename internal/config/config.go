package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Cookie          string
	DeviceID        string
	SessionToken    string
	UserSessionID   string
	UserAgent       string
	ProxyURL        string
	SelectionIDs    []int64
	SelectionNames  map[int64]string
	DeliveryMode    string
	Domain          string
	RequestTimeout  int
	PageLimit       int
	RetryAttempts   int
}

func Load(envPath string) (*Config, error) {
	if err := loadDotEnv(envPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read %s: %w", envPath, err)
	}

	cfg := &Config{
		Cookie:         os.Getenv("LENTA_COOKIE"),
		DeviceID:       os.Getenv("LENTA_DEVICE_ID"),
		SessionToken:   os.Getenv("LENTA_SESSION_TOKEN"),
		UserSessionID:  os.Getenv("LENTA_USER_SESSION_ID"),
		UserAgent:      getEnvDefault("LENTA_USER_AGENT", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36"),
		ProxyURL:       os.Getenv("PROXY_URL"),
		DeliveryMode:   getEnvDefault("LENTA_DELIVERY_MODE", "pickup"),
		Domain:         getEnvDefault("LENTA_DOMAIN", "moscow"),
		RequestTimeout: getEnvInt("REQUEST_TIMEOUT_SECONDS", 30),
		PageLimit:      getEnvInt("LENTA_PAGE_LIMIT", 40),
		RetryAttempts:  getEnvInt("RETRY_ATTEMPTS", 3),
	}

	ids, err := parseIDs(os.Getenv("LENTA_SELECTION_IDS"))
	if err != nil {
		return nil, fmt.Errorf("LENTA_SELECTION_IDS: %w", err)
	}
	cfg.SelectionIDs = ids
	cfg.SelectionNames = parseNames(os.Getenv("LENTA_SELECTION_NAMES"))

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	missing := []string{}
	if c.Cookie == "" {
		missing = append(missing, "LENTA_COOKIE")
	}
	if c.DeviceID == "" {
		missing = append(missing, "LENTA_DEVICE_ID")
	}
	if c.SessionToken == "" {
		missing = append(missing, "LENTA_SESSION_TOKEN")
	}
	if c.UserSessionID == "" {
		missing = append(missing, "LENTA_USER_SESSION_ID")
	}
	if len(c.SelectionIDs) == 0 {
		missing = append(missing, "LENTA_SELECTION_IDS")
	}
	if c.ProxyURL == "" {
		missing = append(missing, "PROXY_URL")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}
	return nil
}

func parseNames(raw string) map[int64]string {
	out := map[int64]string{}
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		colon := strings.IndexByte(pair, ':')
		if colon <= 0 {
			continue
		}
		id, err := strconv.ParseInt(strings.TrimSpace(pair[:colon]), 10, 64)
		if err != nil {
			continue
		}
		out[id] = strings.TrimSpace(pair[colon+1:])
	}
	return out
}

func parseIDs(raw string) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid id %q: %w", p, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		eq := strings.IndexByte(trimmed, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:eq])
		val := strings.TrimSpace(trimmed[eq+1:])
		val = strings.Trim(val, `"'`)
		if _, present := os.LookupEnv(key); present {
			continue
		}
		_ = os.Setenv(key, val)
	}
	return sc.Err()
}
