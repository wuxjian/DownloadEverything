package ai

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// PreprocessHTML 预处理HTML，用goquery提取关键内容给AI分析
func PreprocessHTML(htmlContent string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return htmlContent
	}

	// 移除script、style等无关标签
	doc.Find("script, style, noscript, iframe, svg, nav, footer, header").Remove()

	// 提取所有链接
	var links []string
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		text := strings.TrimSpace(s.Text())
		if text == "" {
			text = "[no text]"
		}
		links = append(links, text+" -> "+href)
	})

	// 获取纯文本
	bodyText := strings.TrimSpace(doc.Find("body").Text())

	// 组合：文本摘要 + 链接列表
	var result strings.Builder
	if len(bodyText) > 5000 {
		result.WriteString(bodyText[:5000])
	} else {
		result.WriteString(bodyText)
	}
	result.WriteString("\n\n--- 页面链接 ---\n")
	for _, l := range links {
		result.WriteString(l + "\n")
	}

	return result.String()
}

// ExtractDirectLinks 用goquery提取直接的下载链接（不经过AI）
func ExtractDirectLinks(htmlContent, baseURL string) []DownloadLink {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil
	}

	base, _ := url.Parse(baseURL)
	var links []DownloadLink
	fileExts := map[string]bool{
		".exe": true, ".zip": true, ".rar": true, ".7z": true,
		".mp4": true, ".mkv": true, ".avi": true, ".mov": true,
		".mp3": true, ".flac": true, ".wav": true,
		".pdf": true, ".doc": true, ".docx": true, ".xls": true,
		".txt": true, ".epub": true, ".mobi": true,
		".iso": true, ".dmg": true, ".deb": true, ".rpm": true,
		".apk": true, ".tar": true, ".gz": true,
	}

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		// 解析URL
		parsedURL, err := url.Parse(href)
		if err != nil {
			return
		}

		// 相对URL转绝对
		if !parsedURL.IsAbs() && base != nil {
			parsedURL = base.ResolveReference(parsedURL)
			href = parsedURL.String()
		}

		// 检查是否是磁力链接
		if strings.HasPrefix(href, "magnet:?") {
			text := strings.TrimSpace(s.Text())
			if text == "" {
				text = "Magnet Link"
			}
			links = append(links, DownloadLink{
				Name: text,
				URL:  href,
				Type: "magnet",
			})
			return
		}

		// 检查文件扩展名
		path := strings.ToLower(parsedURL.Path)
		for ext := range fileExts {
			if strings.HasSuffix(path, ext) {
				text := strings.TrimSpace(s.Text())
				if text == "" {
					text = parsedURL.Path
				}
				links = append(links, DownloadLink{
					Name: text,
					URL:  href,
					Type: ext[1:],
				})
				return
			}
		}
	})

	return links
}
