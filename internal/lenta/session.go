package lenta

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type Session struct {
	SessionToken  string
	DeviceID      string
	UserSessionID string
	Cookie        string
	UserAgent     string
}

type BootstrapOptions struct {
	ProxyURL  string
	UserAgent string
	Wait      time.Duration
}

func Bootstrap(ctx context.Context, opts BootstrapOptions) (*Session, error) {
	if opts.Wait == 0 {
		opts.Wait = 12 * time.Second
	}
	if opts.UserAgent == "" {
		opts.UserAgent = defaultUA
	}

	execOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", "new"),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.ProxyServer(opts.ProxyURL),
		chromedp.UserAgent(opts.UserAgent),
	)
	alloc, cancelAlloc := chromedp.NewExecAllocator(ctx, execOpts...)
	defer cancelAlloc()
	browserCtx, cancelBrowser := chromedp.NewContext(alloc)
	defer cancelBrowser()

	var (
		mu           sync.Mutex
		sessionToken string
	)
	chromedp.ListenTarget(browserCtx, func(ev any) {
		e, ok := ev.(*network.EventResponseReceived)
		if !ok || !strings.Contains(e.Response.URL, "/api/rest/sessionGet") {
			return
		}
		rid := e.RequestID
		go func() {
			c, cancel := context.WithTimeout(browserCtx, 5*time.Second)
			defer cancel()
			_ = chromedp.Run(c, chromedp.ActionFunc(func(c context.Context) error {
				body, err := network.GetResponseBody(rid).Do(c)
				if err != nil {
					return nil
				}
				var r struct {
					Body struct{ SessionToken string }
				}
				if json.Unmarshal(body, &r) == nil && r.Body.SessionToken != "" {
					mu.Lock()
					sessionToken = r.Body.SessionToken
					mu.Unlock()
				}
				return nil
			}))
		}()
	})

	var cookies []*network.Cookie
	err := chromedp.Run(browserCtx,
		network.Enable(),
		chromedp.Navigate(baseURL+"/"),
		chromedp.Sleep(opts.Wait),
		chromedp.ActionFunc(func(c context.Context) error {
			cc, err := network.GetCookies().Do(c)
			if err != nil {
				return err
			}
			cookies = cc
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("chromedp: %w", err)
	}

	s := &Session{
		Cookie:    buildCookieHeader(cookies),
		UserAgent: opts.UserAgent,
	}
	for _, c := range cookies {
		switch c.Name {
		case "UserSessionId":
			s.UserSessionID = c.Value
		case "Utk_DvcGuid":
			s.DeviceID = c.Value
		}
	}
	mu.Lock()
	s.SessionToken = sessionToken
	mu.Unlock()

	if s.SessionToken == "" {
		return nil, fmt.Errorf("bootstrap: sessionGet did not return SessionToken (Qrator probably blocked the page)")
	}
	if s.DeviceID == "" {
		return nil, fmt.Errorf("bootstrap: cookie Utk_DvcGuid missing")
	}
	if s.UserSessionID == "" {
		return nil, fmt.Errorf("bootstrap: cookie UserSessionId missing")
	}
	return s, nil
}

func buildCookieHeader(cookies []*network.Cookie) string {
	parts := make([]string, 0, len(cookies))
	for _, c := range cookies {
		if c.Domain != "" && !strings.Contains(c.Domain, "lenta.com") {
			continue
		}
		parts = append(parts, c.Name+"="+c.Value)
	}
	return strings.Join(parts, "; ")
}

const defaultUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36"
