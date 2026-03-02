package main

// --- HTTP API 请求/响应类型 ---

// SuccessResponse 通用成功响应
type SuccessResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data"`
	Message string `json:"message,omitempty"`
}

// ErrorResponse 通用错误响应
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details any    `json:"details,omitempty"`
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Keyword string `json:"keyword" binding:"required"`
	Count   int    `json:"count"`
}

// PuhuoRequest 铺货请求
type PuhuoRequest struct {
	URL string `json:"url" binding:"required"`
}

// --- MCP 工具中间类型 ---

// MCPToolResult MCP 工具返回结果
type MCPToolResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"is_error,omitempty"`
}

// MCPContent MCP 内容项
type MCPContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}
