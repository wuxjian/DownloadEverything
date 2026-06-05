package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client OpenAI兼容API客户端
type Client struct {
	Endpoint string
	Model    string
	APIKey   string
	HTTP     *http.Client
}

// NewClient 创建AI客户端
func NewClient(endpoint, model, apiKey string) *Client {
	return &Client{
		Endpoint: endpoint,
		Model:    model,
		APIKey:   apiKey,
		HTTP:     &http.Client{},
	}
}

// Message 聊天消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Chat 发送聊天请求
func (c *Client) Chat(messages []Message, temperature float64) (string, error) {
	reqBody := ChatRequest{
		Model:       c.Model,
		Messages:    messages,
		Temperature: temperature,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", c.Endpoint+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("AI请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI API返回 %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("解析AI响应失败: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("AI未返回任何结果")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// ChatJSON 发送聊天并解析JSON响应
func (c *Client) ChatJSON(messages []Message, result interface{}) error {
	content, err := c.Chat(messages, 0.1)
	if err != nil {
		return err
	}

	// 尝试提取JSON (处理markdown代码块)
	jsonStr := extractJSON(content)
	if err := json.Unmarshal([]byte(jsonStr), result); err != nil {
		return fmt.Errorf("解析AI JSON响应失败: %w, 原始内容: %s", err, content)
	}
	return nil
}

// extractJSON 从可能的markdown代码块中提取JSON
func extractJSON(s string) string {
	// 简单处理：查找第一个 [ 或 { 和最后一个 ] 或 }
	start := -1
	end := -1
	for i, c := range s {
		if c == '[' || c == '{' {
			if start == -1 {
				start = i
			}
		}
		if c == ']' || c == '}' {
			end = i
		}
	}
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}
