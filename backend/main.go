package main

import (
	"log"
	"balance-monitor/internal/config"
	"balance-monitor/internal/database"
	"balance-monitor/internal/router"
	"balance-monitor/internal/scheduler"
)

func main() {
	// 初始化配置
	cfg := config.Load()

	// 初始化数据库
	db, err := database.Init(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer db.Close()

	// 启动定时任务
	scheduler.Start(db)

	// 启动 Web 服务
	r := router.Setup(db)
	log.Printf("服务器启动在端口 %s", cfg.ServerPort)
	if err := r.Run(cfg.ServerPort); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

