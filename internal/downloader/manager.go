package downloader

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"download-everything/internal/config"
	dbmod "download-everything/internal/database"
)

// Manager 下载任务管理器
type Manager struct {
	db         *sql.DB
	cfg        *config.Config
	store      *dbmod.TaskStore
	httpDL     *HTTPDownloader
	active     map[string]context.CancelFunc
	mu         sync.RWMutex
	semaphore  chan struct{}
	broadcastCh chan *ProgressEvent
}

// ProgressEvent 进度广播事件
type ProgressEvent struct {
	TaskID string
	P      *Progress
}

// NewManager 创建下载管理器
func NewManager(db *sql.DB, cfg *config.Config) *Manager {
	m := &Manager{
		db:          db,
		cfg:         cfg,
		store:       dbmod.NewTaskStore(db),
		active:      make(map[string]context.CancelFunc),
		semaphore:   make(chan struct{}, cfg.MaxConcurrent),
		broadcastCh: make(chan *ProgressEvent, 100),
	}
	m.httpDL = NewHTTPDownloader(cfg.ThreadsPerFile, cfg.ProxyURL, cfg.MaxRetries, cfg.RetryInterval, func(taskID string, p *Progress) {
		// 更新数据库
		m.store.UpdateProgress(taskID, p.Downloaded, p.Speed, p.Progress, p.Status)
		// 广播进度
		select {
		case m.broadcastCh <- &ProgressEvent{TaskID: taskID, P: p}:
		default:
		}
	})

	// 确保下载目录存在
	os.MkdirAll(cfg.DownDir, 0755)

	return m
}

// Subscribe 订阅进度事件
func (m *Manager) Subscribe() <-chan *ProgressEvent {
	return m.broadcastCh
}

// AddTask 添加下载任务并启动
func (m *Manager) AddTask(url, name, source string, headers map[string]string, cookies []*http.Cookie) (*dbmod.Task, error) {
	task := &dbmod.Task{
		URL:     url,
		Name:    name,
		Status:  "pending",
		Source:  source,
		Headers: headers,
		Cookies: cookies,
	}

	if err := m.store.CreateTask(task); err != nil {
		return nil, fmt.Errorf("保存任务失败: %w", err)
	}

	go m.startDownload(task)
	return task, nil
}

// startDownload 启动下载
func (m *Manager) startDownload(task *dbmod.Task) {
	// 获取并发信号量
	m.semaphore <- struct{}{}
	defer func() { <-m.semaphore }()

	ctx, cancel := context.WithCancel(context.Background())
	m.mu.Lock()
	m.active[task.ID] = cancel
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		delete(m.active, task.ID)
		m.mu.Unlock()
		cancel()
	}()

	// 构建文件路径
	filePath := filepath.Join(m.cfg.DownDir, task.Name)
	if task.FilePath != "" {
		filePath = task.FilePath
	}
	m.store.UpdateFilePath(task.ID, filePath)

	dlTask := &DownloadTask{
		ID:       task.ID,
		URL:      task.URL,
		Name:     task.Name,
		FilePath: filePath,
		Headers:  task.Headers,
		Cookies:  task.Cookies,
	}

	m.store.UpdateStatus(task.ID, "downloading", "")
	err := m.httpDL.Download(ctx, dlTask)
	if err != nil {
		if ctx.Err() == context.Canceled {
			m.store.UpdateStatus(task.ID, "paused", "")
		} else {
			log.Printf("下载失败 [%s]: %v", task.Name, err)
			m.store.UpdateStatus(task.ID, "failed", err.Error())
		}
	} else {
		m.store.UpdateStatus(task.ID, "done", "")
	}
}

// PauseTask 暂停任务
func (m *Manager) PauseTask(id string) error {
	m.mu.RLock()
	cancel, ok := m.active[id]
	m.mu.RUnlock()
	if ok {
		cancel()
	}
	return m.store.UpdateStatus(id, "paused", "")
}

// ResumeTask 恢复任务
func (m *Manager) ResumeTask(id string) error {
	task, err := m.store.GetTask(id)
	if err != nil {
		return err
	}
	if task.Status != "paused" {
		return fmt.Errorf("任务状态不允许恢复: %s", task.Status)
	}
	task.Status = "pending"
	go m.startDownload(task)
	return nil
}

// CancelTask 取消任务
func (m *Manager) CancelTask(id string) error {
	m.mu.RLock()
	cancel, ok := m.active[id]
	m.mu.RUnlock()
	if ok {
		cancel()
	}
	return m.store.UpdateStatus(id, "failed", "用户取消")
}

// DeleteTask 删除任务
func (m *Manager) DeleteTask(id string) error {
	m.mu.RLock()
	cancel, ok := m.active[id]
	m.mu.RUnlock()
	if ok {
		cancel()
	}
	return m.store.DeleteTask(id)
}

// GetTask 获取任务详情
func (m *Manager) GetTask(id string) (*dbmod.Task, error) {
	return m.store.GetTask(id)
}

// ListTasks 获取所有任务
func (m *Manager) ListTasks() ([]*dbmod.Task, error) {
	return m.store.ListTasks()
}

// CountByStatus 统计任务状态
func (m *Manager) CountByStatus() (map[string]int, error) {
	return m.store.CountByStatus()
}

// Shutdown 关闭管理器
func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, cancel := range m.active {
		cancel()
	}
}

// UpdateDownloaderConfig 热更新下载器配置（代理、重试等）
func (m *Manager) UpdateDownloaderConfig(cfg *config.Config) {
	m.httpDL = NewHTTPDownloader(cfg.ThreadsPerFile, cfg.ProxyURL, cfg.MaxRetries, cfg.RetryInterval, func(taskID string, p *Progress) {
		m.store.UpdateProgress(taskID, p.Downloaded, p.Speed, p.Progress, p.Status)
		select {
		case m.broadcastCh <- &ProgressEvent{TaskID: taskID, P: p}:
		default:
		}
	})
	log.Printf("[下载] 下载器配置已更新: 代理=%s, 重试=%d次, 间隔=%ds", cfg.ProxyURL, cfg.MaxRetries, cfg.RetryInterval)
}

// MarshalHeaders 序列化header（给外部调用用）
func MarshalHeaders(h map[string]string) string {
	data, _ := json.Marshal(h)
	return string(data)
}
