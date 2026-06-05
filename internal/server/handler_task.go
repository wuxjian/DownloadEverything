package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"

	"download-everything/internal/service"

	"github.com/gin-gonic/gin"
)

// createTask 创建下载任务
func (h *AppHandler) createTask(c *gin.Context) {
	var req service.CreateTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if req.Name == "" {
		req.Name = path.Base(req.URL)
	}

	task, err := h.Manager.AddTask(req.URL, req.Name, req.Source, req.Headers, req.Cookies)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": task.ID, "status": task.Status})
}

// listTasks 获取任务列表
func (h *AppHandler) listTasks(c *gin.Context) {
	tasks, err := h.Manager.ListTasks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tasks == nil {
		c.JSON(http.StatusOK, gin.H{"tasks": []interface{}{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

// getTask 获取任务详情
func (h *AppHandler) getTask(c *gin.Context) {
	id := c.Param("id")
	task, err := h.Manager.GetTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	c.JSON(http.StatusOK, task)
}

// pauseTask 暂停任务
func (h *AppHandler) pauseTask(c *gin.Context) {
	id := c.Param("id")
	if err := h.Manager.PauseTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// resumeTask 恢复任务
func (h *AppHandler) resumeTask(c *gin.Context) {
	id := c.Param("id")
	if err := h.Manager.ResumeTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// retryTask 重试失败的任务
func (h *AppHandler) retryTask(c *gin.Context) {
	id := c.Param("id")
	if err := h.Manager.RetryTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// deleteTask 删除任务
func (h *AppHandler) deleteTask(c *gin.Context) {
	id := c.Param("id")
	if err := h.Manager.DeleteTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// clearTasks 清空所有已结束的历史任务
func (h *AppHandler) clearTasks(c *gin.Context) {
	if err := h.Manager.DeleteAllTasks(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// taskEvents SSE进度推送
func (h *AppHandler) taskEvents(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ch, unsubscribe := h.Manager.Subscribe()
	defer unsubscribe()

	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-ch:
			if !ok {
				return false
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			return true
		case <-time.After(30 * time.Second):
			fmt.Fprintf(w, ": keepalive\n\n")
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

