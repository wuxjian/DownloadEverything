package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// TavilyClient Tavily搜索API客户端
type TavilyClient struct {
	APIKey string
	HTTP   *http.Client
}

// NewTavilyClient 创建Tavily客户端
func NewTavilyClient(apiKey string) *TavilyClient {
	return &TavilyClient{
		APIKey: apiKey,
		HTTP:   &http.Client{},
	}
}

// SearchResult 搜索结果
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
	Score   float64 `json:"score"`
}

// SearchResponse Tavily搜索响应
type SearchResponse struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
}

// searchRequest Tavily搜索请求体
type searchRequest struct {
	APIKey       string `json:"api_key"`
	Query        string `json:"query"`
	MaxResults   int    `json:"max_results"`
	SearchDepth  string `json:"search_depth"`
	IncludeAnswer bool  `json:"include_answer"`
}

// Search 执行搜索
func (t *TavilyClient) Search(query string, maxResults int) (*SearchResponse, error) {
	if t.APIKey == "" {
		return nil, fmt.Errorf("Tavily API Key 未配置，请在设置中配置")
	}

	reqBody := searchRequest{
		APIKey:        t.APIKey,
		Query:         query,
		MaxResults:    maxResults,
		SearchDepth:   "basic",
		IncludeAnswer: true,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.tavily.com/search", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Tavily搜索请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Tavily API返回 %d: %s", resp.StatusCode, string(body))
	}

	var searchResp SearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("解析Tavily响应失败: %w", err)
	}

	return &searchResp, nil
}
