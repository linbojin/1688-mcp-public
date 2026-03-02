package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"
)

// AppServer 应用服务器，管理 HTTP + MCP 双协议
type AppServer struct {
	alibabaService *AlibabaService
	mcpServer      *mcp.Server
	router         *gin.Engine
	httpServer     *http.Server
}

// NewAppServer 创建应用服务器
func NewAppServer(alibabaService *AlibabaService) *AppServer {
	appServer := &AppServer{
		alibabaService: alibabaService,
	}

	// 初始化 MCP Server
	appServer.mcpServer = InitMCPServer(appServer)

	return appServer
}

// Start 启动 HTTP 服务器
func (s *AppServer) Start(addr string) error {
	s.router = setupRoutes(s)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	// 启动服务
	go func() {
		logrus.Infof("1688-MCP 服务启动: http://%s", addr)
		logrus.Infof("  MCP 端点: http://%s/mcp", addr)
		logrus.Infof("  REST API: http://%s/api/v1/", addr)
		logrus.Infof("  健康检查: http://%s/health", addr)

		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("正在关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		logrus.Warnf("等待连接关闭超时，强制退出: %v", err)
	} else {
		logrus.Info("服务器已优雅关闭")
	}

	return nil
}
