package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// SerperClient Serper.dev 搜索API客户端
type SerperClient struct {
	APIKey string
	HTTP   *http.Client
}

// NewSerperClient 创建Serper客户端
func NewSerperClient(apiKey string) *SerperClient {
	return &SerperClient{
		APIKey: apiKey,
		HTTP:   &http.Client{},
	}
}

// serperSearchResponse Serper.dev 搜索响应
type serperSearchResponse struct {
	Organic []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
	} `json:"organic"`
}

// serperSearchRequest Serper搜索请求体
type serperSearchRequest struct {
	Query string `json:"q"`
	HL    string `json:"hl"`
	Page  int    `json:"page"`
}

// Search 执行搜索，返回统一格式的搜索结果
func (s *SerperClient) Search(query string, maxResults int) (*SearchResponse, error) {
	if s.APIKey == "" {
		return nil, fmt.Errorf("Serper API Key 未配置，请在设置中配置")
	}

	reqBody := serperSearchRequest{
		Query: query,
		HL:    "zh-cn",
		Page:  1,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://google.serper.dev/search", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-KEY", s.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Serper搜索请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Serper API返回 %d: %s", resp.StatusCode, string(body))
	}

	var serperResp serperSearchResponse
	if err := json.Unmarshal(body, &serperResp); err != nil {
		return nil, fmt.Errorf("解析Serper响应失败: %w", err)
	}

	// 转换为统一格式
	results := make([]SearchResult, 0, len(serperResp.Organic))
	for _, item := range serperResp.Organic {
		results = append(results, SearchResult{
			Title:   item.Title,
			URL:     item.Link,
			Content: item.Snippet,
		})
	}

	return &SearchResponse{
		Query:   query,
		Results: results,
	}, nil
}
