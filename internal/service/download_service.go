package service

import (
	"download-everything/internal/downloader"
	"net/http"
	"path"
)

// DownloadService 下载业务逻辑
type DownloadService struct {
	Manager *downloader.Manager
}

// NewDownloadService 创建下载服务
func NewDownloadService(mgr *downloader.Manager) *DownloadService {
	return &DownloadService{Manager: mgr}
}

// CreateTaskReq 创建任务请求
type CreateTaskReq struct {
	URL     string            `json:"url" binding:"required"`
	Name    string            `json:"name"`
	Source  string            `json:"source"`
	Headers map[string]string `json:"headers"`
	Cookies []*http.Cookie    `json:"cookies"`
}

// AddTask 添加下载任务
func (s *DownloadService) AddTask(req *CreateTaskReq) (*downloader.Progress, error) {
	name := req.Name
	if name == "" {
		// 从URL提取文件名
		name = path.Base(req.URL)
		if name == "" || name == "." || name == "/" {
			name = "download"
		}
	}
	source := req.Source
	if source == "" {
		source = "manual"
	}

	task, err := s.Manager.AddTask(req.URL, name, source, req.Headers, req.Cookies)
	if err != nil {
		return nil, err
	}

	return &downloader.Progress{
		Status: task.Status,
		Total:  task.FileSize,
	}, nil
}
