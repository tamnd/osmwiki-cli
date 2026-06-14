// Package osmwiki is the library behind the osmwiki command line:
// the HTTP client, request shaping, and the typed data models for the
// OpenStreetMap wiki fetched via the MediaWiki API.
package osmwiki

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const DefaultUserAgent = "Mozilla/5.0 (compatible; osmwiki-cli/0.1; +https://github.com/tamnd/osmwiki-cli)"

var htmlTagRe = regexp.MustCompile("<[^>]+>")

// Config holds constructor parameters.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://wiki.openstreetmap.org",
		UserAgent: DefaultUserAgent,
		Rate:      500 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the OpenStreetMap wiki MediaWiki API.
type Client struct {
	cfg        Config
	httpClient *http.Client
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client with the given config.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}
}

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		b, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return b, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get: %w", lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// Search searches OpenStreetMap wiki pages by keyword.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Page, error) {
	if limit <= 0 {
		limit = 20
	}
	apiURL := fmt.Sprintf(
		"%s/w/api.php?action=query&list=search&srsearch=%s&srlimit=%d&format=json",
		c.cfg.BaseURL, url.QueryEscape(query), limit,
	)
	raw, err := c.get(ctx, apiURL)
	if err != nil {
		return nil, err
	}
	var resp wireSearchResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	pages := make([]Page, 0, len(resp.Query.Search))
	for i, w := range resp.Query.Search {
		pages = append(pages, wireToPage(w, i+1))
	}
	return pages, nil
}

// GetPage fetches a single page extract by title.
// Returns nil, nil when the page does not exist (pageid == -1).
func (c *Client) GetPage(ctx context.Context, title string) (*PageDetail, error) {
	apiURL := fmt.Sprintf(
		"%s/w/api.php?action=query&prop=extracts&exintro&titles=%s&format=json",
		c.cfg.BaseURL, url.QueryEscape(title),
	)
	raw, err := c.get(ctx, apiURL)
	if err != nil {
		return nil, err
	}
	var resp wireExtractResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("decode page response: %w", err)
	}
	for _, wp := range resp.Query.Pages {
		if wp.PageID == -1 {
			return nil, nil
		}
		d := wireToPageDetail(wp)
		return &d, nil
	}
	return nil, nil
}

func pageURL(title string) string {
	return fmt.Sprintf("https://wiki.openstreetmap.org/wiki/%s", strings.ReplaceAll(title, " ", "_"))
}

func stripHTML(s string) string {
	return htmlTagRe.ReplaceAllString(s, "")
}

func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

func wireToPage(w wireSearchPage, rank int) Page {
	snippet := truncateRunes(stripHTML(w.Snippet), 120)
	updated := ""
	if len(w.Timestamp) >= 10 {
		updated = w.Timestamp[:10]
	}
	return Page{
		Rank:      rank,
		ID:        w.PageID,
		Title:     w.Title,
		Snippet:   snippet,
		WordCount: w.WordCount,
		Updated:   updated,
		URL:       pageURL(w.Title),
	}
}

func wireToPageDetail(w wireExtractPage) PageDetail {
	extract := truncateRunes(stripHTML(w.Extract), 500)
	return PageDetail{
		ID:      w.PageID,
		Title:   w.Title,
		Extract: extract,
		URL:     pageURL(w.Title),
	}
}
