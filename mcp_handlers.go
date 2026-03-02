package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"
)

// handleCheckLogin 处理 check_login 工具调用
func (s *AppServer) handleCheckLogin(ctx context.Context) *MCPToolResult {
	logrus.Info("MCP: 检查登录状态")

	status, err := s.alibabaService.CheckLogin(ctx)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "检查登录状态失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("序列化结果失败: %v", err),
			}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: string(data),
		}},
	}
}

// handleRefreshCookies 处理 refresh_cookies 工具调用
func (s *AppServer) handleRefreshCookies(ctx context.Context) *MCPToolResult {
	logrus.Info("MCP: 重新加载 cookies")

	result, err := s.alibabaService.RefreshCookies(ctx)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "重新加载 cookies 失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("序列化结果失败: %v", err),
			}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: string(data),
		}},
	}
}

// handleSearch1688 处理 search_1688 工具调用
func (s *AppServer) handleSearch1688(ctx context.Context, args map[string]any) *MCPToolResult {
	logrus.Info("MCP: 搜索1688商品")

	keyword, ok := args["keyword"].(string)
	if !ok || keyword == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "搜索失败: 缺少 keyword 参数",
			}},
			IsError: true,
		}
	}

	count := 20
	if raw, ok := args["count"]; ok {
		switch v := raw.(type) {
		case float64:
			count = int(v)
		case string:
			if parsed, err := strconv.Atoi(v); err == nil {
				count = parsed
			}
		case int:
			count = v
		}
	}

	products, err := s.alibabaService.Search1688(ctx, keyword, count)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "搜索失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	data, err := json.MarshalIndent(products, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("序列化结果失败: %v", err),
			}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: string(data),
		}},
	}
}

// handlePuhuo 处理 puhuo 工具调用
func (s *AppServer) handlePuhuo(ctx context.Context, args map[string]any) *MCPToolResult {
	logrus.Info("MCP: 铺货商品")

	url, ok := args["url"].(string)
	if !ok || url == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "铺货失败: 缺少 url 参数",
			}},
			IsError: true,
		}
	}

	result, err := s.alibabaService.ProductPuhuo(ctx, url)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "铺货失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("序列化结果失败: %v", err),
			}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: string(data),
		}},
	}
}
