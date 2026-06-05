package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"download-everything/internal/ai"

	"github.com/gin-gonic/gin"
)

// aiSearch AI搜索（SSE流式返回各步骤进度）
func (h *AppHandler) aiSearch(c *gin.Context) {
	var req struct {
		Query string `json:"query" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入搜索关键词"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	type sseData struct {
		Step    int         `json:"step"`
		Total   int         `json:"total"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
		Done    bool        `json:"done"`
		Error   string      `json:"error,omitempty"`
	}

	sendSSE := func(d sseData) {
		data, _ := json.Marshal(d)
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		c.Writer.(http.Flusher).Flush()
	}

	cb := func(step, totalSteps int, message string, data interface{}) {
		sendSSE(sseData{Step: step, Total: totalSteps, Message: message, Data: data})
	}

	result, err := h.AISvc.Search(ctx, req.Query, cb)
	if err != nil {
		sendSSE(sseData{Done: true, Error: err.Error()})
		return
	}

	sendSSE(sseData{Step: 4, Total: 4, Message: fmt.Sprintf("搜索完成，找到 %d 个下载链接", len(result.Links)), Data: result, Done: true})
}

// aiParseURL 解析单个URL
func (h *AppHandler) aiParseURL(c *gin.Context) {
	var req struct {
		URL string `json:"url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入URL"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	links, err := h.AISvc.ParseURL(ctx, req.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"links": links})
}

// aiDownload 将AI搜索到的链接加入下载队列
func (h *AppHandler) aiDownload(c *gin.Context) {
	var req struct {
		Links []ai.DownloadLink `json:"links" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	var ids []string
	for _, link := range req.Links {
		task, err := h.Manager.AddTask(link.URL, link.Name, "ai_search", nil, nil)
		if err != nil {
			continue
		}
		ids = append(ids, task.ID)
	}

	c.JSON(http.StatusOK, gin.H{"task_ids": ids, "count": len(ids)})
}
