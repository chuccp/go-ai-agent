package tool

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// WebSearch is an internet search tool.
// For Claude models, type=web_search_20260209 is handled natively by the API, bypassing Execute.
// For other models (OpenAI/Gemini etc.), Execute performs actual search via DuckDuckGo.
type WebSearch struct{}

var WebSearchType = "web_search_20260209"

func (t *WebSearch) Definition() Definition {
	return Definition{
		Type:        WebSearchType,
		Name:        "web_search",
		Description: "Search the internet for the latest information. Use when you need real-time data, news, or information beyond the model's knowledge cutoff date.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Search query",
				},
			},
			"required": []string{"query"},
		},
	}
}

var (
	reLink    = regexp.MustCompile(`class="result__a"[^>]*href="([^"]*)"[^>]*>([^<]+)</a>`)
	reSnippet = regexp.MustCompile(`class="result__snippet"[^>]*>((?s:.*?))</a>`)
)

func (t *WebSearch) Execute(call Call) (string, error) {
	query := ""
	if args := parseArgs(call.Arguments); args != nil {
		query = args["query"]
	}
	if query == "" {
		return "", fmt.Errorf("search query is required")
	}

	results, err := searchDuckDuckGo(query)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}
	if results == "" {
		return fmt.Sprintf("No results found for \"%s\"", query), nil
	}
	return results, nil
}

func searchDuckDuckGo(query string) (string, error) {
	u := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; go-ai-agent/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return "", err
	}

	return parseResults(string(body)), nil
}

func parseResults(html string) string {
	links := reLink.FindAllStringSubmatch(html, -1)
	snippets := reSnippet.FindAllStringSubmatch(html, -1)

	if len(links) == 0 {
		// Regex parsing failed. As a fallback, return a cleaned-up raw HTML
		// fragment so the caller still gets useful content instead of nothing.
		return fallbackRawHTML(html)
	}

	var b strings.Builder
	count := min(len(links), 5) // top 5 results
	for i := 0; i < count; i++ {
		title := cleanHTML(links[i][2])
		link := cleanURL(links[i][1])

		b.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, title))
		b.WriteString(fmt.Sprintf("   %s\n", link))

		if i < len(snippets) {
			snippet := cleanHTML(snippets[i][1])
			snippet = strings.TrimSpace(snippet)
			if len(snippet) > 300 {
				snippet = snippet[:300] + "..."
			}
			if snippet != "" {
				b.WriteString(fmt.Sprintf("   %s\n", snippet))
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

// fallbackRawHTML is used when the structured regex parsing fails. It returns a
// best-effort cleaned text fragment derived from the raw HTML so that the caller
// still receives some content rather than an empty string.
func fallbackRawHTML(html string) string {
	// If there is genuinely no result-related content, return empty.
	if !strings.Contains(html, "result") && !strings.Contains(html, "Result") {
		return ""
	}
	cleaned := cleanHTML(html)
	cleaned = strings.TrimSpace(cleaned)
	if len(cleaned) > 2000 {
		cleaned = cleaned[:2000] + "..."
	}
	if cleaned == "" {
		return ""
	}
	return "Search results (raw fallback):\n" + cleaned
}

func cleanHTML(s string) string {
	s = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#x27;", "'")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func cleanURL(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	// Strip DuckDuckGo redirect wrapper
	if strings.Contains(s, "//duckduckgo.com/l/?uddg=") {
		if u, err := url.Parse(s); err == nil {
			if raw := u.Query().Get("uddg"); raw != "" {
				if decoded, err := url.QueryUnescape(raw); err == nil {
					return decoded
				}
			}
		}
	}
	return s
}

func parseArgs(argsJSON string) map[string]string {
	// Use encoding/json for robust parsing instead of a fragile regex.
	result := make(map[string]string)
	if strings.TrimSpace(argsJSON) == "" {
		return result
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &raw); err != nil {
		// Fall back to empty map on parse error; callers handle missing keys.
		return result
	}
	for k, v := range raw {
		switch val := v.(type) {
		case string:
			result[k] = val
		case nil:
			result[k] = ""
		default:
			result[k] = fmt.Sprintf("%v", val)
		}
	}
	return result
}
