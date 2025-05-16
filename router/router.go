package router

import (
	"dify-upload-workflow/config"
	"dify-upload-workflow/controller"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

// SetupRouter 设置路由
func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 跨域中间件
	r.Use(corsMiddleware())

	// 请求日志中间件
	r.Use(requestLoggerMiddleware())

	// 增强的健康检查路由
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":       "ok",
			"version":      "1.0.0",
			"goVersion":    runtime.Version(),
			"numCPU":       runtime.NumCPU(),
			"numGoroutine": runtime.NumGoroutine(),
			"serverTime":   time.Now().Format(time.RFC3339),
			"config": gin.H{
				"maxUploadFiles": config.Config.MaxUploadFiles,
				"maxFileSize":    config.Config.MaxFileSize,
				"defaultTimeout": config.Config.DefaultApiTimeout,
			},
		})
	})

	// API分组
	dify := r.Group("/dify")
	{
		// 单文件 URL 工作流
		dify.POST("/fileSingle/workflow", controller.SingleFileHandler)

		// 多文件 URL 工作流
		dify.POST("/files/workflow", controller.MultiFilesHandler)

		// 单文件 form-data 工作流
		dify.POST("/fileSingle/formdata/workflow", controller.SingleFileFormHandler)

		// 多文件 form-data 工作流
		dify.POST("/files/formdata/workflow", controller.MultiFilesFormHandler)

		// 仅上传文件到dify
		dify.POST("/upload/url", controller.UploadURLFileHandler)

		// 异步请求状态查询
		dify.GET("/async/:requestID", controller.QueryAsyncStatus)
	}

	return r
}

// requestLoggerMiddleware 记录请求日志的中间件
func requestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()

		// 处理请求
		c.Next()

		// 结束时间
		endTime := time.Now()

		// 执行时间
		latencyTime := endTime.Sub(startTime)

		// 请求方式
		reqMethod := c.Request.Method

		// 请求路由
		reqUri := c.Request.RequestURI

		// 状态码
		statusCode := c.Writer.Status()

		// 请求IP
		clientIP := c.ClientIP()

		// 日志格式
		log.Printf("[GIN] %s | %3d | %13v | %15s | %s",
			reqMethod,
			statusCode,
			latencyTime,
			clientIP,
			reqUri,
		)
	}
}

// corsMiddleware 处理跨域请求的中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
