package ai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SearchPipeline AI搜索流水线
type SearchPipeline struct {
	aiClient     *Client
	tavilyClient *TavilyClient
	serperClient *SerperClient
}

// NewSearchPipeline 创建搜索流水线
func NewSearchPipeline(aiClient *Client, tavilyClient *TavilyClient, serperClient *SerperClient) *SearchPipeline {
	return &SearchPipeline{
		aiClient:     aiClient,
		tavilyClient: tavilyClient,
		serperClient: serperClient,
	}
}

// StepCallback 步骤回调，用于报告进度
type StepCallback func(step int, totalSteps int, message string, data interface{})

// PipelineResult 流水线最终结果
type PipelineResult struct {
	Query     string          `json:"query"`
	Links     []DownloadLink  `json:"links"`
	RawPages  int             `json:"raw_pages"`
}

// DownloadLink 提取的下载链接
type DownloadLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Size string `json:"size"`
	Type string `json:"type"`
}

// Run 执行完整搜索流水线
// Step1: 搜索引擎搜索 -> Step2: AI筛选 -> Step3: HTTP抓取 -> Step4: AI提取链接
func (p *SearchPipeline) Run(ctx context.Context, query string, cb StepCallback) (*PipelineResult, error) {
	totalSteps := 4

	// === Step1: 搜索引擎搜索 ===
	if cb != nil {
		cb(1, totalSteps, "正在搜索: "+query, nil)
	}

	// 检查可用的搜索引擎
	useTavily := p.tavilyClient != nil && p.tavilyClient.APIKey != ""
	useSerper := p.serperClient != nil && p.serperClient.APIKey != ""

	if !useTavily && !useSerper {
		return nil, fmt.Errorf("未配置任何搜索引擎，请在设置中配置 Tavily 或 Serper API Key")
	}

	// 执行搜索
	var allResults []SearchResult
	var searchMsg string

	if useTavily && useSerper {
		// 两个都配置，并发执行
		type searchResult struct {
			results []SearchResult
			err     error
			source  string
		}

		ch := make(chan searchResult, 2)

		go func() {
			resp, err := p.tavilyClient.Search(query, 20)
			if err != nil {
				ch <- searchResult{nil, err, "Tavily"}
			} else {
				ch <- searchResult{resp.Results, nil, "Tavily"}
			}
		}()

		go func() {
			resp, err := p.serperClient.Search(query, 20)
			if err != nil {
				ch <- searchResult{nil, err, "Serper"}
			} else {
				ch <- searchResult{resp.Results, nil, "Serper"}
			}
		}()

		// 收集结果
		urlSet := make(map[string]bool)
		for i := 0; i < 2; i++ {
			res := <-ch
			if res.err != nil {
				// 一个失败不影响另一个
				continue
			}
			for _, r := range res.results {
				if !urlSet[r.URL] {
					urlSet[r.URL] = true
					allResults = append(allResults, r)
				}
			}
		}
		searchMsg = fmt.Sprintf("合并搜索完成，找到 %d 个结果", len(allResults))
	} else if useTavily {
		// 只用 Tavily
		resp, err := p.tavilyClient.Search(query, 20)
		if err != nil {
			return nil, fmt.Errorf("Step1 Tavily搜索失败: %w", err)
		}
		allResults = resp.Results
		searchMsg = fmt.Sprintf("Tavily 搜索完成，找到 %d 个结果", len(allResults))
	} else {
		// 只用 Serper
		resp, err := p.serperClient.Search(query, 20)
		if err != nil {
			return nil, fmt.Errorf("Step1 Serper搜索失败: %w", err)
		}
		allResults = resp.Results
		searchMsg = fmt.Sprintf("Serper 搜索完成，找到 %d 个结果", len(allResults))
	}

	if cb != nil {
		cb(1, totalSteps, searchMsg, allResults)
	}

	if len(allResults) == 0 {
		return &PipelineResult{Query: query}, nil
	}

	// === Step2: AI筛选最相关网页 ===
	if cb != nil {
		cb(2, totalSteps, "AI正在分析搜索结果，筛选最相关网页...", nil)
	}
	selectedURLs, err := p.filterResults(ctx, allResults, query)
	if err != nil {
		return nil, fmt.Errorf("Step2 AI筛选失败: %w", err)
	}
	if cb != nil {
		cb(2, totalSteps, fmt.Sprintf("AI筛选出 %d 个相关网页", len(selectedURLs)), selectedURLs)
	}

	if len(selectedURLs) == 0 {
		return &PipelineResult{Query: query}, nil
	}

	// === Step3: HTTP抓取网页内容 ===
	if cb != nil {
		cb(3, totalSteps, "正在抓取网页内容...", nil)
	}
	pages := make(map[string]string)
	for _, url := range selectedURLs {
		content, err := fetchPage(ctx, url)
		if err != nil {
			continue
		}
		pages[url] = content
	}
	if cb != nil {
		cb(3, totalSteps, fmt.Sprintf("成功抓取 %d 个网页", len(pages)), nil)
	}

	if len(pages) == 0 {
		return &PipelineResult{Query: query}, nil
	}

	// === Step4: AI提取下载链接 ===
	if cb != nil {
		cb(4, totalSteps, "AI正在分析网页内容，提取下载链接...", nil)
	}
	links, err := p.extractLinks(ctx, pages, query)
	if err != nil {
		return nil, fmt.Errorf("Step4 AI提取链接失败: %w", err)
	}
	if cb != nil {
		cb(4, totalSteps, fmt.Sprintf("找到 %d 个下载链接", len(links)), links)
	}

	// 补充：用 ExtractDirectLinks 直接从 HTML 中匹配文件扩展名链接，补全 AI 可能遗漏的
	linkSet := make(map[string]bool)
	for _, l := range links {
		linkSet[l.URL] = true
	}
	var extraLinks []DownloadLink
	for url, content := range pages {
		directLinks := ExtractDirectLinks(content, url)
		for _, dl := range directLinks {
			if !linkSet[dl.URL] {
				linkSet[dl.URL] = true
				extraLinks = append(extraLinks, dl)
			}
		}
	}
	if len(extraLinks) > 0 {
		links = append(links, extraLinks...)
	}

	return &PipelineResult{
		Query:    query,
		Links:    links,
		RawPages: len(pages),
	}, nil
}

// filterResults AI筛选最相关的搜索结果
func (p *SearchPipeline) filterResults(ctx context.Context, results []SearchResult, query string) ([]string, error) {
	var resultDesc strings.Builder
	for i, r := range results {
		resultDesc.WriteString(fmt.Sprintf("%d. [%s] %s\n   摘要: %s\n\n", i+1, r.URL, r.Title, r.Content))
	}

	messages := []Message{
		{
			Role: "system",
			Content: `你是一个资源搜索助手。用户搜索了某个资源，下面是搜索引擎返回的结果。
请分析这些结果，选出最可能包含下载链接的网页URL（最多选5个）。
注意：下载链接不限于 .exe/.zip 等二进制文件，.txt 小说、电子书、文档等文本类资源同样视为有效的下载链接。
只返回JSON数组，格式如：["url1", "url2", ...]
如果没有合适的结果，返回空数组 []。`,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("搜索关键词: %s\n\n搜索结果:\n%s", query, resultDesc.String()),
		},
	}

	var urls []string
	if err := p.aiClient.ChatJSON(messages, &urls); err != nil {
		return nil, err
	}
	return urls, nil
}

// extractLinks AI从网页内容中提取下载链接
func (p *SearchPipeline) extractLinks(ctx context.Context, pages map[string]string, query string) ([]DownloadLink, error) {
	var allLinks []DownloadLink

	for url, content := range pages {
		// 限制内容长度，避免超出AI token限制
		if len(content) > 15000 {
			content = content[:15000]
		}

		messages := []Message{
			{
				Role: "system",
				Content: `你是一个资源下载链接提取助手。请从给定的网页内容中提取所有可能的下载链接。
提取规则：
1. 寻找直接的下载链接（.exe, .zip, .rar, .7z, .mp4, .mkv, .pdf, .mp3, .txt, .epub, .mobi 等文件链接）
2. 特别注意文本类资源：.txt 小说、电子书（.epub, .mobi）、文档（.pdf, .doc, .docx）等
3. 寻找下载按钮或下载区域的链接
4. 寻找网盘分享链接（如百度网盘、阿里云盘、Google Drive等）
5. 寻找磁力链接（magnet:?开头的链接）

返回JSON数组，格式如：
[{"name": "文件名", "url": "下载链接", "size": "文件大小(如有)", "type": "文件类型"}]
如果没有找到下载链接，返回空数组 []。`,
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("用户搜索: %s\n网页URL: %s\n\n网页内容:\n%s", query, url, content),
			},
		}

		var links []DownloadLink
		if err := p.aiClient.ChatJSON(messages, &links); err != nil {
			continue
		}
		allLinks = append(allLinks, links...)
	}

	return allLinks, nil
}

// fetchPage 抓取网页内容
func fetchPage(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// 限制读取大小 (最多1MB)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// ParseURL 解析单个URL提取下载链接（用于手动输入URL场景）
func (p *SearchPipeline) ParseURL(ctx context.Context, url string) ([]DownloadLink, error) {
	content, err := fetchPage(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("抓取网页失败: %w", err)
	}

	pages := map[string]string{url: content}
	return p.extractLinks(ctx, pages, "")
}
