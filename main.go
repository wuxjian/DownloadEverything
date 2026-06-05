package main

import (
	"download-everything/internal/config"
	"download-everything/internal/database"
	"download-everything/internal/downloader"
	"download-everything/internal/server"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 先连接数据库
	db, err := database.Init("download.db")
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer db.Close()

	// 从数据库加载配置，首次运行自动写入默认值
	store := database.NewSettingsStore(db)
	cfg, err := config.LoadFromDB(store)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化下载管理器
	mgr := downloader.NewManager(db, cfg)
	defer mgr.Shutdown()

	// 启动HTTP服务器
	router := server.NewRouter(db, cfg, mgr, WebFS)
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Download Everything 启动于 http://localhost%s", addr)

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := router.Run(addr); err != nil {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	<-quit
	log.Println("正在关闭服务...")
}
