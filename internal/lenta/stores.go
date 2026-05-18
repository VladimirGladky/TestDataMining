package lenta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Store struct {
	ID          int64  `json:"id"`
	Alias       string `json:"alias"`
	Title       string `json:"title"`
	AddressFull string `json:"addressFull"`
}

type storesSearchResponse struct {
	Items []Store `json:"items"`
}

func (c *Client) FindStore(ctx context.Context, query string) (*Store, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		baseURL+"/api-gateway/v1/stores/pickup/search", bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, err
	}
	c.applyHeaders(req)
	req.Header.Set("referer", baseURL+"/")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stores/pickup/search: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stores/pickup/search status %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	var out storesSearchResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode stores: %w", err)
	}

	q := strings.ToLower(strings.TrimSpace(query))
	matches := make([]Store, 0, 4)
	for _, s := range out.Items {
		if matchesStore(s, q) {
			matches = append(matches, s)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no store matched %q (searched %d stores). Examples: %s",
			query, len(out.Items), sampleAddresses(out.Items, 3))
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("query %q is ambiguous (%d matches), first 10: %s — please refine",
			query, len(matches), listAddresses(matches[:min(10, len(matches))]))
	}
}

func matchesStore(s Store, q string) bool {
	return strings.Contains(strings.ToLower(s.Alias), q) ||
		strings.Contains(strings.ToLower(s.Title), q) ||
		strings.Contains(strings.ToLower(s.AddressFull), q)
}

func listAddresses(ss []Store) string {
	parts := make([]string, 0, len(ss))
	for _, s := range ss {
		parts = append(parts, fmt.Sprintf("%s (%s)", s.AddressFull, s.Alias))
	}
	return strings.Join(parts, "; ")
}

func sampleAddresses(ss []Store, n int) string {
	if len(ss) < n {
		n = len(ss)
	}
	return listAddresses(ss[:n])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *Client) SelectPickupStore(ctx context.Context, storeID int64) error {
	url := fmt.Sprintf("%s/api-gateway/v1/stores/pickup/%d", baseURL, storeID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, nil)
	if err != nil {
		return err
	}
	c.applyHeaders(req)
	req.Header.Set("referer", baseURL+"/")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("select store: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("select store %d: status %d: %s", storeID, resp.StatusCode, truncate(string(body), 300))
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
