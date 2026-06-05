package server

import (
	"encoding/json"
	"net/http"

	"download-everything/internal/config"
	"download-everything/internal/database"

	"github.com/gin-gonic/gin"
)

// getSettings 获取配置
func (h *AppHandler) getSettings(c *gin.Context) {
	c.JSON(http.StatusOK, h.Cfg)
}

// updateSettings 更新配置
func (h *AppHandler) updateSettings(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.Cfg.Update(updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新配置失败: " + err.Error()})
		return
	}

	// 保存配置到数据库
	store := database.NewSettingsStore(h.DB)
	if err := config.SaveToDB(store, h.Cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存配置失败: " + err.Error()})
		return
	}

	// 刷新AI客户端
	h.AISvc.RefreshClient(h.Cfg)

	// 热更新下载器配置（代理、重试等）
	h.Manager.UpdateDownloaderConfig(h.Cfg)

	// 返回更新后的配置（脱敏）
	safe := map[string]interface{}{}
	data, _ := json.Marshal(h.Cfg)
	json.Unmarshal(data, &safe)
	// 脱敏API Key
	if key, ok := safe["ai_key"].(string); ok && len(key) > 8 {
		safe["ai_key"] = key[:4] + "****" + key[len(key)-4:]
	}
	if key, ok := safe["tavily_key"].(string); ok && len(key) > 8 {
		safe["tavily_key"] = key[:4] + "****" + key[len(key)-4:]
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "config": safe})
}