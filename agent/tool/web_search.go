package tool

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

func init() {
	Register(&WebSearch{})
}

// WebSearch 联网搜索工具。
// 对于 Claude 模型，type=web_search_20260209 由 API 原生处理，不走 Execute。
// 对于其他模型（OpenAI/Gemini 等），Execute 通过 DuckDuckGo 执行实际搜索。
type WebSearch struct{}

var WebSearchType = "web_search_20260209"

func (t *WebSearch) Definition() Definition {
	return Definition{
		Type:        WebSearchType,
		Name:        "web_search",
		Description: "搜索互联网获取最新信息。当需要查找实时数据、新闻、或模型知识截止日期之后的信息时使用。",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "搜索关键词",
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
		return fmt.Sprintf("搜索「%s」未找到相关结果", query), nil
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
		return ""
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
	// Simple JSON string parser — avoid heavy dependency for a flat string map
	result := make(map[string]string)
	re := regexp.MustCompile(`"([^"]+)"\s*:\s*"((?:[^"\\]|\\.)*)"`)
	matches := re.FindAllStringSubmatch(argsJSON, -1)
	for _, m := range matches {
		result[m[1]] = strings.ReplaceAll(m[2], `\"`, `"`)
	}
	return result
}
