package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/linbojin/1688-mcp-public/alibaba"
)

// AlibabaService 1688 核心业务逻辑
type AlibabaService struct {
	mu     sync.RWMutex
	client *http.Client
}

// NewAlibabaService 创建服务实例
func NewAlibabaService() (*AlibabaService, error) {
	client, err := alibaba.NewHTTPClient()
	if err != nil {
		return nil, err
	}
	logrus.Info("HTTP client 初始化完成（带 cookies）")
	return &AlibabaService{client: client}, nil
}

// Search1688 搜索商品，count 为返回条数（默认 20）
func (s *AlibabaService) Search1688(ctx context.Context, keyword string, count int) ([]alibaba.Product, error) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()
	return alibaba.Search1688(ctx, client, keyword, count)
}

// ProductPuhuo 铺货单个商品
func (s *AlibabaService) ProductPuhuo(ctx context.Context, url string) (*alibaba.PuhuoResult, error) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()
	return alibaba.ProductPuhuo(ctx, client, url)
}

// CheckLogin 验证当前 cookies 是否有效
func (s *AlibabaService) CheckLogin(ctx context.Context) (*alibaba.LoginStatus, error) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()
	return alibaba.CheckLogin(ctx, client)
}

// RefreshCookies 从文件重新加载 cookies 并重建 HTTP client
// 若需重新登录，请先运行 1688-login 工具
func (s *AlibabaService) RefreshCookies(ctx context.Context) (*alibaba.RefreshResult, error) {
	logrus.Info("重新加载 cookies...")

	newClient, err := alibaba.NewHTTPClient()
	if err != nil {
		msg := fmt.Sprintf("重新加载 cookies 失败: %v", err)
		return &alibaba.RefreshResult{Success: false, Message: msg}, nil
	}

	cookies, _ := alibaba.LoadCookies()

	s.mu.Lock()
	s.client = newClient
	s.mu.Unlock()

	logrus.Infof("cookies 重新加载完成，共 %d 条", len(cookies))
	return &alibaba.RefreshResult{
		Success:     true,
		CookieCount: len(cookies),
		Message:     "cookies 重新加载成功（如需重新登录，请运行 1688-login 工具）",
	}, nil
}
