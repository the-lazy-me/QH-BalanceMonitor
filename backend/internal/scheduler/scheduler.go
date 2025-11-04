package scheduler

import (
	"database/sql"
	"log"
	"sync"
	"time"

	"balance-monitor/internal/services"
	"github.com/robfig/cron/v3"
)

var (
	cronInstance       *cron.Cron
	entryID            cron.EntryID
	currentCronExpr    string // 当前使用的 cron 表达式
	mu                 sync.Mutex
	db                 *sql.DB
)

func Start(database *sql.DB) {
	db = database
	cronInstance = cron.New()

	// 加载监控设置并启动定时任务
	updateCronJob()

	cronInstance.Start()
	log.Println("定时任务已启动")

	// 每分钟检查一次是否需要更新 cron 表达式
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			updateCronJob()
		}
	}()
}

func updateCronJob() {
	mu.Lock()
	defer mu.Unlock()

	// 获取数据库中的 cron 表达式
	var cronExpression string
	err := db.QueryRow("SELECT cron_expression FROM monitor_settings WHERE id = 1").Scan(&cronExpression)
	if err != nil {
		log.Printf("获取 cron 表达式失败: %v", err)
		cronExpression = "@hourly" // 使用默认值
	}

	// 检查 cron 表达式是否发生变化
	if cronExpression == currentCronExpr && entryID != 0 {
		// 表达式未改变，无需更新
		return
	}

	log.Printf("检测到 cron 表达式变化: '%s' -> '%s'，正在更新定时任务...", currentCronExpr, cronExpression)

	// 移除旧的定时任务
	if entryID != 0 {
		cronInstance.Remove(entryID)
	}

	// 添加新的定时任务
	newEntryID, err := cronInstance.AddFunc(cronExpression, func() {
		log.Println("开始执行定时余额检查...")
		results, err := services.CheckAllProviders(db)
		if err != nil {
			log.Printf("余额检查失败: %v", err)
			return
		}
		log.Printf("余额检查完成，检查了 %d 个供应商", len(results))
		for _, result := range results {
			log.Printf("  - %s: 余额 %.2f", result.ProviderName, result.Balance)
		}
	})

	if err != nil {
		log.Printf("添加定时任务失败: %v，使用默认 cron 表达式", err)
		// 如果添加失败，尝试使用默认值
		cronExpression = "@hourly"
		newEntryID, _ = cronInstance.AddFunc(cronExpression, func() {
			log.Println("开始执行定时余额检查...")
			results, err := services.CheckAllProviders(db)
			if err != nil {
				log.Printf("余额检查失败: %v", err)
				return
			}
			log.Printf("余额检查完成，检查了 %d 个供应商", len(results))
			for _, result := range results {
				log.Printf("  - %s: 余额 %.2f", result.ProviderName, result.Balance)
			}
		})
	}

	entryID = newEntryID
	currentCronExpr = cronExpression
	log.Printf("定时任务已更新，cron 表达式: %s", cronExpression)
}

