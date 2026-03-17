package main

import (
	"balance-monitor/internal/config"
	"balance-monitor/internal/database"
	"balance-monitor/internal/router"
	"balance-monitor/internal/scheduler"
	"balance-monitor/internal/services"
	"log"
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

	if err := services.MarkInterruptedConsumptionBackfillJobs(db); err != nil {
		log.Fatalf("历史回补任务中断修复失败: %v", err)
	}

	// 启动定时任务
	scheduler.Start(db)

	// 启动 Web 服务
	r := router.Setup(db)
	log.Printf("服务器启动在端口 %s", cfg.ServerPort)
	if err := r.Run(cfg.ServerPort); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
