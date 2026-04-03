package main

import (
	"context"
	"cursor2api-go/config"
	"cursor2api-go/handlers"
	"cursor2api-go/middleware"
	"cursor2api-go/models"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// 设置日志级别和 GIN 模式
	if cfg.Debug {
		logrus.SetLevel(logrus.DebugLevel)
		gin.SetMode(gin.DebugMode)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
		gin.SetMode(gin.ReleaseMode)
	}
	gin.DisableConsoleColor()

	// 创建路由器
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.ErrorHandler())
	if cfg.Debug {
		router.Use(gin.Logger())
	}

	// 创建处理器
	handler := handlers.NewHandler(cfg)

	// 注册路由
	setupRoutes(router, handler, cfg)

	// 创建HTTP服务器（设置读写超时）
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Timeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Timeout) * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	printStartupBanner(cfg)

	// 启动服务器
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Server failed: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logrus.Fatalf("Server forced to shutdown: %v", err)
	}
	logrus.Info("Server exited gracefully")
}

func setupRoutes(router *gin.Engine, handler *handlers.Handler, cfg *config.Config) {
	// 健康检查
	router.GET("/health", handler.Health)

	// API文档
	router.GET("/", handler.ServeDocs)

	// API v1
	v1 := router.Group("/v1")
	{
		v1.GET("/models", handler.ListModels)
		v1.POST("/chat/completions", middleware.AuthRequired(cfg.APIKey), handler.ChatCompletions)
	}
}

// printStartupBanner 打印启动横幅
func printStartupBanner(cfg *config.Config) {
	banner := `
╔══════════════════════════════════════════════════════════════╗
║                      Cursor2API Server                       ║
╚══════════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
	fmt.Printf("🚀  服务地址:  http://localhost:%d\n", cfg.Port)
	fmt.Printf("📚  API 文档:  http://localhost:%d/\n", cfg.Port)
	fmt.Printf("💊  健康检查:  http://localhost:%d/health\n", cfg.Port)
	fmt.Printf("🔑  API 密钥:  %s\n", cfg.MaskedAPIKey())

	modelList := cfg.GetModels()
	fmt.Printf("\n🤖  支持模型 (%d 个):\n", len(modelList))

	// 按Provider分组
	providers := make(map[string][]string)
	for _, modelID := range modelList {
		if mc, exists := models.GetModelConfig(modelID); exists {
			providers[mc.Provider] = append(providers[mc.Provider], modelID)
		} else {
			providers["Other"] = append(providers["Other"], modelID)
		}
	}

	for _, provider := range []string{"Anthropic", "Google", "OpenAI", "Other"} {
		if ms, ok := providers[provider]; ok {
			fmt.Printf("   %s:  %s\n", provider, strings.Join(ms, ", "))
		}
	}

	if cfg.Debug {
		fmt.Println("\n🐛  调试模式:  已启用")
	}
	fmt.Println("\n✨  服务已启动，按 Ctrl+C 停止")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
