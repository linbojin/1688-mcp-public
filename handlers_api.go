package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// --- 响应工具函数 ---

func respondSuccess(c *gin.Context, data any, message string) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

func respondError(c *gin.Context, statusCode int, code, message string, details any) {
	c.JSON(statusCode, ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	})
}

// --- REST API Handlers ---

// loginStatusHandler 检查登录状态
func (s *AppServer) loginStatusHandler(c *gin.Context) {
	status, err := s.alibabaService.CheckLogin(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, "CHECK_LOGIN_FAILED",
			"检查登录状态失败", err.Error())
		return
	}
	respondSuccess(c, status, "")
}

// refreshCookiesHandler 重新加载 cookies
func (s *AppServer) refreshCookiesHandler(c *gin.Context) {
	result, err := s.alibabaService.RefreshCookies(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, "REFRESH_FAILED",
			"重新加载 cookies 失败", err.Error())
		return
	}
	respondSuccess(c, result, result.Message)
}

// searchHandler 搜索商品
func (s *AppServer) searchHandler(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	products, err := s.alibabaService.Search1688(c.Request.Context(), req.Keyword, req.Count)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "SEARCH_FAILED",
			"搜索失败", err.Error())
		return
	}
	respondSuccess(c, products, "搜索成功")
}

// puhuoHandler 铺货单个商品
func (s *AppServer) puhuoHandler(c *gin.Context) {
	var req PuhuoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	result, err := s.alibabaService.ProductPuhuo(c.Request.Context(), req.URL)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "PUHUO_FAILED",
			"铺货失败", err.Error())
		return
	}
	respondSuccess(c, result, "铺货完成")
}
