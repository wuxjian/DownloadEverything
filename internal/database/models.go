package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Task 下载任务模型
type Task struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	URL        string            `json:"url"`
	FilePath   string            `json:"file_path"`
	FileSize   int64             `json:"file_size"`
	Downloaded int64             `json:"downloaded"`
	Speed      int64             `json:"speed"`
	Status     string            `json:"status"`
	Progress   float64           `json:"progress"`
	Source     string            `json:"source"`
	Headers    map[string]string `json:"headers"`
	Cookies    []*http.Cookie    `json:"cookies"`
	ErrorMsg   string            `json:"error_msg"`
	CreatedAt  time.Time         `json:"created_at"`
	FinishedAt *time.Time        `json:"finished_at"`
}

// SearchHistory AI搜索历史
type SearchHistory struct {
	ID        string    `json:"id"`
	Query     string    `json:"query"`
	Results   string    `json:"results"`
	ModelUsed string    `json:"model_used"`
	CreatedAt time.Time `json:"created_at"`
}

// TaskStore 任务数据库操作
type TaskStore struct {
	DB *sql.DB
}

// NewTaskStore 创建任务存储
func NewTaskStore(db *sql.DB) *TaskStore {
	return &TaskStore{DB: db}
}

// CreateTask 创建下载任务
func (s *TaskStore) CreateTask(t *Task) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	t.CreatedAt = time.Now()

	headersJSON, _ := json.Marshal(t.Headers)
	cookiesJSON, _ := json.Marshal(t.Cookies)

	_, err := s.DB.Exec(`
		INSERT INTO tasks (id, name, url, file_path, file_size, downloaded, speed, status, progress, source, headers, cookies, error_msg, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Name, t.URL, t.FilePath, t.FileSize, t.Downloaded, t.Speed,
		t.Status, t.Progress, t.Source, string(headersJSON), string(cookiesJSON),
		t.ErrorMsg, t.CreatedAt)
	return err
}

// GetTask 获取单个任务
func (s *TaskStore) GetTask(id string) (*Task, error) {
	row := s.DB.QueryRow("SELECT id, name, url, file_path, file_size, downloaded, speed, status, progress, source, headers, cookies, error_msg, created_at, finished_at FROM tasks WHERE id = ?", id)
	return scanTask(row)
}

// ListTasks 获取所有任务
func (s *TaskStore) ListTasks() ([]*Task, error) {
	rows, err := s.DB.Query("SELECT id, name, url, file_path, file_size, downloaded, speed, status, progress, source, headers, cookies, error_msg, created_at, finished_at FROM tasks ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		t, err := scanTaskRows(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// UpdateProgress 更新任务进度
func (s *TaskStore) UpdateProgress(id string, downloaded, speed int64, progress float64, status string) error {
	_, err := s.DB.Exec("UPDATE tasks SET downloaded = ?, speed = ?, progress = ?, status = ? WHERE id = ?",
		downloaded, speed, progress, status, id)
	return err
}

// UpdateStatus 更新任务状态
func (s *TaskStore) UpdateStatus(id, status, errMsg string) error {
	var finishedAt *time.Time
	if status == "done" || status == "failed" {
		now := time.Now()
		finishedAt = &now
		_, err := s.DB.Exec("UPDATE tasks SET status = ?, error_msg = ?, finished_at = ? WHERE id = ?",
			status, errMsg, finishedAt, id)
		return err
	}
	_, err := s.DB.Exec("UPDATE tasks SET status = ?, error_msg = ? WHERE id = ?", status, errMsg, id)
	return err
}

// UpdateFilePath 更新文件路径
func (s *TaskStore) UpdateFilePath(id, filePath string) error {
	_, err := s.DB.Exec("UPDATE tasks SET file_path = ? WHERE id = ?", filePath, id)
	return err
}

// UpdateFileSize 更新文件大小
func (s *TaskStore) UpdateFileSize(id string, size int64) error {
	_, err := s.DB.Exec("UPDATE tasks SET file_size = ? WHERE id = ?", size, id)
	return err
}

// DeleteTask 删除任务
func (s *TaskStore) DeleteTask(id string) error {
	_, err := s.DB.Exec("DELETE FROM tasks WHERE id = ?", id)
	return err
}

// CountByStatus 统计各状态任务数
func (s *TaskStore) CountByStatus() (map[string]int, error) {
	rows, err := s.DB.Query("SELECT status, COUNT(*) FROM tasks GROUP BY status")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		rows.Scan(&status, &count)
		result[status] = count
	}
	return result, nil
}

// --- 搜索历史 ---

// SearchHistoryStore 搜索历史存储
type SearchHistoryStore struct {
	DB *sql.DB
}

// NewSearchHistoryStore 创建搜索历史存储
func NewSearchHistoryStore(db *sql.DB) *SearchHistoryStore {
	return &SearchHistoryStore{DB: db}
}

// Create 保存搜索记录
func (s *SearchHistoryStore) Create(query, results, model string) error {
	_, err := s.DB.Exec("INSERT INTO search_history (id, query, results, model_used, created_at) VALUES (?, ?, ?, ?, ?)",
		uuid.New().String(), query, results, model, time.Now())
	return err
}

// List 获取搜索历史
func (s *SearchHistoryStore) List(limit int) ([]*SearchHistory, error) {
	rows, err := s.DB.Query("SELECT id, query, results, model_used, created_at FROM search_history ORDER BY created_at DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*SearchHistory
	for rows.Next() {
		h := &SearchHistory{}
		if err := rows.Scan(&h.ID, &h.Query, &h.Results, &h.ModelUsed, &h.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, h)
	}
	return list, nil
}

// --- 辅助函数 ---

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanTask(row *sql.Row) (*Task, error) {
	t := &Task{}
	var headersStr, cookiesStr string
	var finishedAt sql.NullTime

	err := row.Scan(&t.ID, &t.Name, &t.URL, &t.FilePath, &t.FileSize, &t.Downloaded,
		&t.Speed, &t.Status, &t.Progress, &t.Source, &headersStr, &cookiesStr,
		&t.ErrorMsg, &t.CreatedAt, &finishedAt)
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(headersStr), &t.Headers)
	json.Unmarshal([]byte(cookiesStr), &t.Cookies)
	if finishedAt.Valid {
		t.FinishedAt = &finishedAt.Time
	}
	return t, nil
}

func scanTaskRows(rows *sql.Rows) (*Task, error) {
	t := &Task{}
	var headersStr, cookiesStr string
	var finishedAt sql.NullTime

	err := rows.Scan(&t.ID, &t.Name, &t.URL, &t.FilePath, &t.FileSize, &t.Downloaded,
		&t.Speed, &t.Status, &t.Progress, &t.Source, &headersStr, &cookiesStr,
		&t.ErrorMsg, &t.CreatedAt, &finishedAt)
	if err != nil {
		return nil, fmt.Errorf("scan error: %w", err)
	}

	json.Unmarshal([]byte(headersStr), &t.Headers)
	json.Unmarshal([]byte(cookiesStr), &t.Cookies)
	if finishedAt.Valid {
		t.FinishedAt = &finishedAt.Time
	}
	return t, nil
}
