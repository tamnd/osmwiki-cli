package osmwiki

// Page is a search result from the OpenStreetMap wiki.
type Page struct {
	Rank      int    `json:"rank"`
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Snippet   string `json:"snippet"`
	WordCount int    `json:"words"`
	Updated   string `json:"updated"`
	URL       string `json:"url"`
}

// PageDetail is a full page extract from the OpenStreetMap wiki.
type PageDetail struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Extract string `json:"extract"`
	URL     string `json:"url"`
}

// wire types for MediaWiki search API

type wireSearchResponse struct {
	Query wireSearchQuery `json:"query"`
}

type wireSearchQuery struct {
	SearchInfo wireSearchInfo   `json:"searchinfo"`
	Search     []wireSearchPage `json:"search"`
}

type wireSearchInfo struct {
	TotalHits int `json:"totalhits"`
}

type wireSearchPage struct {
	PageID    int    `json:"pageid"`
	Title     string `json:"title"`
	Snippet   string `json:"snippet"`
	Timestamp string `json:"timestamp"`
	Size      int    `json:"size"`
	WordCount int    `json:"wordcount"`
}

// wire types for MediaWiki extract API

type wireExtractResponse struct {
	Query wireExtractQuery `json:"query"`
}

type wireExtractQuery struct {
	Pages map[string]wireExtractPage `json:"pages"`
}

type wireExtractPage struct {
	PageID  int    `json:"pageid"`
	Title   string `json:"title"`
	Extract string `json:"extract"`
}
