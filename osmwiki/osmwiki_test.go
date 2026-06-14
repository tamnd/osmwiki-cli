package osmwiki_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/osmwiki-cli/osmwiki"
)

func searchPayload(pages []map[string]any) []byte {
	b, _ := json.Marshal(map[string]any{
		"query": map[string]any{
			"searchinfo": map[string]any{"totalhits": len(pages)},
			"search":     pages,
		},
	})
	return b
}

func extractPayload(pages map[string]any) []byte {
	b, _ := json.Marshal(map[string]any{
		"query": map[string]any{
			"pages": pages,
		},
	})
	return b
}

func newTestServer(t *testing.T) (*httptest.Server, func(action string, payload []byte)) {
	t.Helper()
	handlers := map[string][]byte{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/w/api.php" {
			http.NotFound(w, r)
			return
		}
		action := r.URL.Query().Get("action")
		payload, ok := handlers[action]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(payload)
	}))
	set := func(action string, payload []byte) {
		handlers[action] = payload
	}
	return srv, set
}

func TestSearch(t *testing.T) {
	payload := searchPayload([]map[string]any{
		{
			"pageid":    1,
			"title":     "Map features",
			"snippet":   "A guide to <b>map</b> features",
			"timestamp": "2024-05-01T12:00:00Z",
			"size":      1000,
			"wordcount": 200,
		},
		{
			"pageid":    2,
			"title":     "Tagging",
			"snippet":   "How tagging works",
			"timestamp": "2024-04-15T08:00:00Z",
			"size":      500,
			"wordcount": 100,
		},
	})

	srv, set := newTestServer(t)
	defer srv.Close()
	set("query", payload)

	cfg := osmwiki.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0

	c := osmwiki.NewClient(cfg)
	pages, err := c.Search(context.Background(), "map", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	if pages[0].Title != "Map features" {
		t.Errorf("title = %q, want %q", pages[0].Title, "Map features")
	}
	if pages[0].Rank != 1 {
		t.Errorf("rank = %d, want 1", pages[0].Rank)
	}
	if pages[0].Updated != "2024-05-01" {
		t.Errorf("updated = %q, want %q", pages[0].Updated, "2024-05-01")
	}
}

func TestSearchLimit(t *testing.T) {
	items := make([]map[string]any, 5)
	for i := range items {
		items[i] = map[string]any{
			"pageid":    i + 1,
			"title":     "Page",
			"snippet":   "snippet",
			"timestamp": "2024-01-01T00:00:00Z",
			"wordcount": 50,
		}
	}
	payload := searchPayload(items)

	srv, set := newTestServer(t)
	defer srv.Close()
	set("query", payload)

	cfg := osmwiki.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0

	c := osmwiki.NewClient(cfg)
	// server returns 5 results; we just check we get all that the server gave us
	pages, err := c.Search(context.Background(), "page", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 5 {
		t.Fatalf("got %d pages, want 5", len(pages))
	}
}

func TestGetPage(t *testing.T) {
	payload := extractPayload(map[string]any{
		"12345": map[string]any{
			"pageid":  12345,
			"title":   "Node",
			"extract": "<p>A <b>node</b> is a basic OSM element.</p>",
		},
	})

	srv, set := newTestServer(t)
	defer srv.Close()
	set("query", payload)

	cfg := osmwiki.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0

	c := osmwiki.NewClient(cfg)
	detail, err := c.GetPage(context.Background(), "Node")
	if err != nil {
		t.Fatal(err)
	}
	if detail == nil {
		t.Fatal("got nil detail, want page")
	}
	if detail.Title != "Node" {
		t.Errorf("title = %q, want %q", detail.Title, "Node")
	}
	if detail.Extract == "" {
		t.Error("extract is empty")
	}
	// HTML should be stripped
	if detail.Extract != "A node is a basic OSM element." {
		t.Errorf("extract = %q", detail.Extract)
	}
}
