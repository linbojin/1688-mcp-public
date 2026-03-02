package alibaba

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type CDPCookie struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain"`
	Path   string `json:"path"`
}

func loadCookieFile() ([]CDPCookie, error) {
	paths := []string{
		"bin/cookies.json",
		filepath.Join(filepath.Dir(os.Args[0]), "cookies.json"),
	}
	if p := os.Getenv("COOKIES_PATH"); p != "" {
		paths = append([]string{p}, paths...)
	}

	var data []byte
	var err error
	for _, p := range paths {
		data, err = os.ReadFile(p)
		if err == nil {
			logrus.Infof("加载 cookies: %s", p)
			break
		}
	}
	if data == nil {
		return nil, fmt.Errorf("cookies 文件未找到，请先运行 1688-login 工具完成登录")
	}

	var cdpCookies []CDPCookie
	if err := json.Unmarshal(data, &cdpCookies); err != nil {
		return nil, fmt.Errorf("解析 cookies 失败: %w", err)
	}
	return cdpCookies, nil
}

func LoadCookies() ([]*http.Cookie, error) {
	cdpCookies, err := loadCookieFile()
	if err != nil {
		return nil, err
	}

	var cookies []*http.Cookie
	for _, c := range cdpCookies {
		// 去掉 value 首尾的引号（CDP 有时带双引号，Go http 会拒绝）
		val := strings.Trim(c.Value, `"`)
		cookies = append(cookies, &http.Cookie{
			Name:   c.Name,
			Value:  val,
			Domain: c.Domain,
			Path:   c.Path,
		})
	}

	logrus.Infof("加载 %d 条 cookies", len(cookies))
	return cookies, nil
}

func LoadCDPCookiesRaw() ([]CDPCookie, error) {
	return loadCookieFile()
}

func NewHTTPClient() (*http.Client, error) {
	cookies, err := LoadCookies()
	if err != nil {
		return nil, err
	}

	jar, _ := cookiejar.New(nil)
	domains := []string{
		"1688.com", "s.1688.com", "detail.1688.com", "air.1688.com",
		"hz-productposting.1688.com", "offer.1688.com",
		"login.taobao.com", "www.taobao.com",
	}
	for _, domain := range domains {
		u, _ := url.Parse("https://" + domain)
		jar.SetCookies(u, cookies)
	}

	return &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}, nil
}
