package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

const (
	defaultWebSearchEndpoint = "https://api.search.brave.com/res/v1/web/search"
	defaultWebTimeout        = 15 * time.Second
	defaultWebFetchMaxBytes  = 256 * 1024
	maxWebFetchBytes         = 1024 * 1024
	maxWebSearchResults      = 20
)

var (
	htmlScriptRe = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	htmlStyleRe  = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	htmlTagRe    = regexp.MustCompile(`(?s)<[^>]+>`)
	htmlSpaceRe  = regexp.MustCompile(`\s+`)
)

type WebSearchInput struct {
	Query      string `json:"query" jsonschema:"required,description=The search query"`
	MaxResults int    `json:"max_results" jsonschema:"description=Optional per-request result limit"`
}

type WebSearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

type WebSearchOutput struct {
	Query   string            `json:"query"`
	Results []WebSearchResult `json:"results"`
}

type webSearchToolImpl struct {
	apiKey     string
	maxResults int
	endpoint   string
	client     *http.Client
}

func (w *webSearchToolImpl) execute(ctx context.Context, input *WebSearchInput) (*WebSearchOutput, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if strings.TrimSpace(w.apiKey) == "" {
		return nil, fmt.Errorf("web_search requires tools.web.search.api_key")
	}

	limit := input.MaxResults
	if limit <= 0 {
		limit = w.maxResults
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > maxWebSearchResults {
		limit = maxWebSearchResults
	}

	u, err := url.Parse(w.endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid search endpoint: %w", err)
	}
	q := u.Query()
	q.Set("q", query)
	q.Set("count", fmt.Sprintf("%d", limit))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", w.apiKey)

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("web search failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var brave struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&brave); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	out := &WebSearchOutput{
		Query:   query,
		Results: make([]WebSearchResult, 0, len(brave.Web.Results)),
	}
	for _, item := range brave.Web.Results {
		out.Results = append(out.Results, WebSearchResult{
			Title:       item.Title,
			URL:         item.URL,
			Description: item.Description,
		})
	}
	return out, nil
}

func NewWebSearchTool(apiKey string, maxResults int) (tool.InvokableTool, error) {
	if maxResults <= 0 {
		maxResults = 5
	}
	impl := &webSearchToolImpl{
		apiKey:     apiKey,
		maxResults: maxResults,
		endpoint:   defaultWebSearchEndpoint,
		client: &http.Client{
			Timeout: defaultWebTimeout,
		},
	}
	return utils.InferTool("web_search", "Search the web for up-to-date information", impl.execute)
}

type WebFetchInput struct {
	URL      string `json:"url" jsonschema:"required,description=The target URL to fetch"`
	MaxBytes int    `json:"max_bytes" jsonschema:"description=Optional maximum response bytes to keep"`
}

type WebFetchOutput struct {
	URL         string `json:"url"`
	Status      int    `json:"status"`
	ContentType string `json:"content_type"`
	Content     string `json:"content"`
	Truncated   bool   `json:"truncated"`
}

type webFetchToolImpl struct {
	client   *http.Client
	maxBytes int
}

func (w *webFetchToolImpl) execute(ctx context.Context, input *WebFetchInput) (*WebFetchOutput, error) {
	rawURL := strings.TrimSpace(input.URL)
	if rawURL == "" {
		return nil, fmt.Errorf("url is required")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported url scheme: %s", parsed.Scheme)
	}

	maxBytes := input.MaxBytes
	if maxBytes <= 0 {
		maxBytes = w.maxBytes
	}
	if maxBytes <= 0 {
		maxBytes = defaultWebFetchMaxBytes
	}
	if maxBytes > maxWebFetchBytes {
		maxBytes = maxWebFetchBytes
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "golem-web-fetch/1.0")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxBytes+1)))
	if err != nil {
		return nil, err
	}

	truncated := false
	if len(body) > maxBytes {
		body = body[:maxBytes]
		truncated = true
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	content := string(body)
	if strings.Contains(contentType, "text/html") {
		content = htmlToText(content)
	}

	out := &WebFetchOutput{
		URL:         rawURL,
		Status:      resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Content:     strings.TrimSpace(content),
		Truncated:   truncated,
	}

	if resp.StatusCode >= 400 {
		return out, fmt.Errorf("web fetch failed with status %d", resp.StatusCode)
	}
	return out, nil
}

func NewWebFetchTool() (tool.InvokableTool, error) {
	impl := &webFetchToolImpl{
		client: &http.Client{
			Timeout: defaultWebTimeout,
		},
		maxBytes: defaultWebFetchMaxBytes,
	}
	return utils.InferTool("web_fetch", "Fetch content from a URL", impl.execute)
}

func htmlToText(input string) string {
	s := htmlScriptRe.ReplaceAllString(input, " ")
	s = htmlStyleRe.ReplaceAllString(s, " ")
	s = htmlTagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = htmlSpaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
