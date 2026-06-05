package downloader

import (
	"context"
	"net/http"
)

// Downloader 下载器接口，支持扩展其他协议
type Downloader interface {
	Download(ctx context.Context, task *DownloadTask) error
	Pause(taskID string) error
	Resume(taskID string) error
	Cancel(taskID string) error
}

// DownloadTask 下载任务信息
type DownloadTask struct {
	ID       string
	URL      string
	Name     string
	FilePath string
	Headers  map[string]string
	Cookies  []*http.Cookie
}

// Progress 下载进度信息
type Progress struct {
	Downloaded int64   `json:"downloaded"`
	Total      int64   `json:"total"`
	Speed      int64   `json:"speed"`
	Progress   float64 `json:"progress"`
	Status     string  `json:"status"`
}

// DefaultHeaders 默认请求头（每个任务自动继承，用户可覆盖）
var DefaultHeaders = map[string]string{
	"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
	"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
	"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
	"Accept-Encoding": "identity", // 下载不使用gzip，便于断点续传
	"Connection":      "keep-alive",
}
