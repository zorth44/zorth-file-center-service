package main

import (
	"log"

	"github.com/zorth44/zorth-file-center-service/config"
	"github.com/zorth44/zorth-file-center-service/internal/handler"
	"github.com/zorth44/zorth-file-center-service/pkg/database"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库
	db, err := database.InitDB(cfg.Database)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	err = database.InitDatabase(db)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	h := handler.NewHandler(db)

	// 设置Gin路由
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8080"}, // 替换为您的前端URL
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Disposition"},
		AllowCredentials: true,
	}))
	h.SetupRoutes(r)

	// 启动服务器
	if err := r.Run(cfg.Server.Address); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}
