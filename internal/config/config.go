package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"download-everything/internal/database"
)

// Config 应用配置
type Config struct {
	Port    int    `json:"port"`
	DBPath  string `json:"db_path"`
	DownDir string `json:"down_dir"` // 默认下载目录

	// AI 配置
	AIEndpoint string `json:"ai_endpoint"`
	AIModel    string `json:"ai_model"`
	AIKey      string `json:"ai_key"`

	// Tavily 搜索
	TavilyKey string `json:"tavily_key"`

	// 下载参数
	MaxConcurrent  int `json:"max_concurrent"`   // 最大并发下载数
	ThreadsPerFile int `json:"threads_per_file"` // 每个文件分片线程数

	// 代理配置
	ProxyURL string `json:"proxy_url"` // 代理地址，如 http://127.0.0.1:7890，留空则不使用

	// 重试配置
	MaxRetries    int `json:"max_retries"`     // 最大重试次数，默认3
	RetryInterval int `json:"retry_interval"` // 重试间隔（秒），默认10
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		Port:           8080,
		DBPath:         "download.db",
		DownDir:        filepath.Join(home, "Downloads", "DownloadEverything"),
		AIEndpoint:     "https://api.openai.com/v1",
		AIModel:        "gpt-4o-mini",
		MaxConcurrent:  5,
		ThreadsPerFile: 4,
		MaxRetries:     3,
		RetryInterval:  10,
	}
}

// Update 更新配置字段
func (c *Config) Update(updates map[string]interface{}) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	for k, v := range updates {
		m[k] = v
	}
	newData, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(newData, c)
}

// settings 表字段名常量
const (
	keyPort           = "port"
	keyDownDir        = "down_dir"
	keyAIEndpoint     = "ai_endpoint"
	keyAIModel        = "ai_model"
	keyAIKey          = "ai_key"
	keyTavilyKey      = "tavily_key"
	keyMaxConcurrent  = "max_concurrent"
	keyThreadsPerFile = "threads_per_file"
	keyProxyURL       = "proxy_url"
	keyMaxRetries     = "max_retries"
	keyRetryInterval  = "retry_interval"
)

// LoadFromDB 从数据库加载配置，没有记录则写入默认值
func LoadFromDB(store *database.SettingsStore) (*Config, error) {
	cfg := DefaultConfig()

	all, err := store.GetAll()
	if err != nil {
		return nil, fmt.Errorf("读取设置失败: %w", err)
	}

	if len(all) == 0 {
		if err := SaveToDB(store, cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	if v, ok := all[keyPort]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Port = n
		}
	}
	if v, ok := all[keyDownDir]; ok {
		cfg.DownDir = v
	}
	if v, ok := all[keyAIEndpoint]; ok {
		cfg.AIEndpoint = v
	}
	if v, ok := all[keyAIModel]; ok {
		cfg.AIModel = v
	}
	if v, ok := all[keyAIKey]; ok {
		cfg.AIKey = v
	}
	if v, ok := all[keyTavilyKey]; ok {
		cfg.TavilyKey = v
	}
	if v, ok := all[keyMaxConcurrent]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MaxConcurrent = n
		}
	}
	if v, ok := all[keyThreadsPerFile]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.ThreadsPerFile = n
		}
	}
	if v, ok := all[keyProxyURL]; ok {
		cfg.ProxyURL = v
	}
	if v, ok := all[keyMaxRetries]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MaxRetries = n
		}
	}
	if v, ok := all[keyRetryInterval]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RetryInterval = n
		}
	}

	return cfg, nil
}

// SaveToDB 将配置保存到数据库
func SaveToDB(store *database.SettingsStore, cfg *Config) error {
	kv := map[string]string{
		keyPort:           strconv.Itoa(cfg.Port),
		keyDownDir:        cfg.DownDir,
		keyAIEndpoint:     cfg.AIEndpoint,
		keyAIModel:        cfg.AIModel,
		keyAIKey:          cfg.AIKey,
		keyTavilyKey:      cfg.TavilyKey,
		keyMaxConcurrent:  strconv.Itoa(cfg.MaxConcurrent),
		keyThreadsPerFile: strconv.Itoa(cfg.ThreadsPerFile),
		keyProxyURL:       cfg.ProxyURL,
		keyMaxRetries:     strconv.Itoa(cfg.MaxRetries),
		keyRetryInterval:  strconv.Itoa(cfg.RetryInterval),
	}
	return store.SetMulti(kv)
}