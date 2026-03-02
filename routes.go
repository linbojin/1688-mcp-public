package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// setupRoutes 配置路由
func setupRoutes(appServer *AppServer) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(errorHandlingMiddleware())
	router.Use(corsMiddleware())

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "1688-mcp",
			"version": "1.0.0",
		})
	})

	// MCP 端点
	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			return appServer.mcpServer
		},
		&mcp.StreamableHTTPOptions{
			JSONResponse: true,
		},
	)
	router.Any("/mcp", gin.WrapH(mcpHandler))
	router.Any("/mcp/*path", gin.WrapH(mcpHandler))

	// REST API 路由
	api := router.Group("/api/v1")
	{
		api.POST("/search", appServer.searchHandler)
		api.POST("/puhuo", appServer.puhuoHandler)

		login := api.Group("/login")
		{
			login.GET("/status", appServer.loginStatusHandler)
			login.POST("/refresh", appServer.refreshCookiesHandler)
		}
	}

	return router
}
