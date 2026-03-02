package main

import (
	"context"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"
)

// InitMCPServer 初始化 MCP 服务器并注册所有工具
func InitMCPServer(appServer *AppServer) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "1688-mcp",
			Version: "1.0.0",
		},
		nil,
	)

	registerTools(server, appServer)

	return server
}

// withPanicRecovery 包装 MCP 工具处理函数，增加 panic 恢复
func withPanicRecovery[T any](
	toolName string,
	handler func(context.Context, *mcp.CallToolRequest, T) (*mcp.CallToolResult, any, error),
) func(context.Context, *mcp.CallToolRequest, T) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args T) (result *mcp.CallToolResult, resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				logrus.WithFields(logrus.Fields{
					"tool":  toolName,
					"panic": r,
				}).Error("MCP 工具执行 panic")

				result = &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{
							Text: fmt.Sprintf("工具 %s 执行时发生内部错误: %v", toolName, r),
						},
					},
					IsError: true,
				}
			}
		}()

		return handler(ctx, req, args)
	}
}

// convertToMCPResult 将内部结果转换为 MCP 结果
func convertToMCPResult(result *MCPToolResult) *mcp.CallToolResult {
	var contents []mcp.Content
	for _, c := range result.Content {
		switch c.Type {
		case "text":
			contents = append(contents, &mcp.TextContent{Text: c.Text})
		}
	}

	return &mcp.CallToolResult{
		Content: contents,
		IsError: result.IsError,
	}
}

func boolPtr(b bool) *bool {
	return &b
}

// registerTools 注册 MCP 工具
func registerTools(server *mcp.Server, appServer *AppServer) {
	// Tool 1: check_login
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "check_login",
			Description: "验证当前 cookies 是否有效（能否访问 1688 登录保护页面）",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: map[string]*jsonschema.Schema{},
			},
			Annotations: &mcp.ToolAnnotations{
				Title:        "检查登录状态",
				ReadOnlyHint: true,
			},
		},
		withPanicRecovery("check_login", func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			result := appServer.handleCheckLogin(ctx)
			return convertToMCPResult(result), nil, nil
		}),
	)

	// Tool 2: refresh_cookies
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "refresh_cookies",
			Description: "从磁盘重新加载 cookies 文件，用于重新登录后刷新会话（需先运行 1688-login 工具）",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: map[string]*jsonschema.Schema{},
			},
			Annotations: &mcp.ToolAnnotations{
				Title:           "刷新Cookies",
				ReadOnlyHint:    false,
				DestructiveHint: boolPtr(true),
			},
		},
		withPanicRecovery("refresh_cookies", func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			result := appServer.handleRefreshCookies(ctx)
			return convertToMCPResult(result), nil, nil
		}),
	)

	// Tool 3: search_1688
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "search_1688",
			Description: "在 1688 搜索代发友好商品（自动过滤包邮+一件代发+48H发货+抖音密文面单）",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"keyword": {
						Type:        "string",
						Description: "搜索关键词",
					},
					"count": {
						Type:        "integer",
						Description: "返回条数，默认 20，最大 20",
					},
				},
				Required: []string{"keyword"},
			},
			Annotations: &mcp.ToolAnnotations{
				Title:        "搜索1688商品",
				ReadOnlyHint: true,
			},
		},
		withPanicRecovery("search_1688", func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			result := appServer.handleSearch1688(ctx, args)
			return convertToMCPResult(result), nil, nil
		}),
	)

	// Tool 4: puhuo
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "puhuo",
			Description: "给定 1688 商品链接，完成一键铺货到抖音小店（浏览器自动化）",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"url": {
						Type:        "string",
						Description: "detail.1688.com 商品链接",
					},
				},
				Required: []string{"url"},
			},
			Annotations: &mcp.ToolAnnotations{
				Title:           "铺货商品",
				ReadOnlyHint:    false,
				DestructiveHint: boolPtr(true),
			},
		},
		withPanicRecovery("puhuo", func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			result := appServer.handlePuhuo(ctx, args)
			return convertToMCPResult(result), nil, nil
		}),
	)

	logrus.Info("MCP 工具注册完成: check_login, refresh_cookies, search_1688, puhuo")
}
