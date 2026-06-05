package server

import (
	"os"
	"os/exec"
	"runtime"

	"github.com/gin-gonic/gin"
)

// openDownloadDir 用系统文件管理器打开下载目录
func (h *AppHandler) openDownloadDir(c *gin.Context) {
	dir := h.Cfg.DownDir
	os.MkdirAll(dir, 0755)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}

	if err := cmd.Start(); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true, "path": dir})
}
