package main

import (
	"dify-upload-workflow/config"
	"dify-upload-workflow/router"
	"log"
	"os"
)

func main() {
	// 检查环境变量
	env := os.Getenv("GO_ENV")
	if env == "" {
		log.Println("未设置GO_ENV环境变量，默认使用开发环境")
	} else {
		log.Printf("当前运行环境: %s", env)
	}

	// 初始化配置
	config.InitConfig()

	// 设置路由
	r := router.SetupRouter()

	// 获取端口
	port := config.GetPort()

	// 启动服务
	log.Printf("服务启动，监听端口：%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
