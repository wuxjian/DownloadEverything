package service

import (
	"context"
	"database/sql"
	"encoding/json"

	"download-everything/internal/ai"
	"download-everything/internal/config"
	dbmod "download-everything/internal/database"
)

// AIService AI搜索业务逻辑
type AIService struct {
	cfg      *config.Config
	pipeline *ai.SearchPipeline
	histStore *dbmod.SearchHistoryStore
}

// NewAIService 创建AI服务
func NewAIService(cfg *config.Config, db *sql.DB) *AIService {
	aiClient := ai.NewClient(cfg.AIEndpoint, cfg.AIModel, cfg.AIKey)
	tavilyClient := ai.NewTavilyClient(cfg.TavilyKey)
	serperClient := ai.NewSerperClient(cfg.SerperKey)
	pipeline := ai.NewSearchPipeline(aiClient, tavilyClient, serperClient)

	return &AIService{
		cfg:       cfg,
		pipeline:  pipeline,
		histStore: dbmod.NewSearchHistoryStore(db),
	}
}

// RefreshClient 刷新AI客户端配置
func (s *AIService) RefreshClient(cfg *config.Config) {
	aiClient := ai.NewClient(cfg.AIEndpoint, cfg.AIModel, cfg.AIKey)
	tavilyClient := ai.NewTavilyClient(cfg.TavilyKey)
	serperClient := ai.NewSerperClient(cfg.SerperKey)
	s.pipeline = ai.NewSearchPipeline(aiClient, tavilyClient, serperClient)
	s.cfg = cfg
}

// Search 执行AI搜索
func (s *AIService) Search(ctx context.Context, query string, cb ai.StepCallback) (*ai.PipelineResult, error) {
	result, err := s.pipeline.Run(ctx, query, cb)
	if err != nil {
		return nil, err
	}

	// 保存搜索历史
	resultsJSON, _ := json.Marshal(result)
	s.histStore.Create(query, string(resultsJSON), s.cfg.AIModel)

	return result, nil
}

// ParseURL 解析单个URL
func (s *AIService) ParseURL(ctx context.Context, url string) ([]ai.DownloadLink, error) {
	return s.pipeline.ParseURL(ctx, url)
}

// GetHistory 获取搜索历史
func (s *AIService) GetHistory(limit int) ([]*dbmod.SearchHistory, error) {
	return s.histStore.List(limit)
}
