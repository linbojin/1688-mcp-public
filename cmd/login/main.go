package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type CDPCookie struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain"`
	Path   string `json:"path"`
}

func main() {
	// 保存路径：与二进制文件同目录的 cookies.json
	cookiesPath := filepath.Join(filepath.Dir(os.Args[0]), "cookies.json")

	fmt.Println("1688 登录工具")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("正在启动浏览器...")

	// 启动可见浏览器（非无头模式）
	l := launcher.New().Headless(false)
	u, err := l.Launch()
	if err != nil {
		fmt.Fprintf(os.Stderr, "启动浏览器失败: %v\n", err)
		os.Exit(1)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "连接浏览器失败: %v\n", err)
		os.Exit(1)
	}
	defer browser.Close()

	// 打开 1688 主页
	page, err := browser.Page(proto.TargetCreateTarget{URL: "https://www.1688.com"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "打开 1688 失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("")
	fmt.Println("请在弹出的浏览器窗口中登录 1688.com")
	fmt.Println("支持手机号登录或支付宝扫码登录")
	fmt.Println("登录成功后程序将自动保存 cookies 并退出")
	fmt.Println("═══════════════════════════════════════")

	// 轮询检测登录状态
	for {
		time.Sleep(500 * time.Millisecond)

		// 检查当前 URL
		info, err := page.Info()
		if err != nil {
			continue
		}

		url := info.URL

		// 如果在登录页面，继续等待
		if strings.Contains(url, "login.taobao.com") ||
			strings.Contains(url, "login.1688.com") ||
			strings.Contains(url, "member.taobao.com/member/login") {
			continue
		}

		// 检查是否已登录（member 页面或主页带有用户信息）
		loggedIn, err := checkLoginState(page)
		if err != nil || !loggedIn {
			continue
		}

		fmt.Println("")
		fmt.Println("✓ 检测到已登录，正在保存 cookies...")

		// 提取 cookies
		cookies, err := page.Cookies([]string{
			"https://www.1688.com",
			"https://detail.1688.com",
			"https://air.1688.com",
			"https://member.1688.com",
			"https://login.taobao.com",
			"https://www.taobao.com",
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "提取 cookies 失败: %v\n", err)
			os.Exit(1)
		}

		// 转换为 CDPCookie 格式
		var cdpCookies []CDPCookie
		for _, c := range cookies {
			cdpCookies = append(cdpCookies, CDPCookie{
				Name:   c.Name,
				Value:  c.Value,
				Domain: c.Domain,
				Path:   c.Path,
			})
		}

		// 保存到文件
		data, err := json.MarshalIndent(cdpCookies, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "序列化 cookies 失败: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(cookiesPath, data, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "保存 cookies 失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ 已保存 %d 条 cookies 到: %s\n", len(cdpCookies), cookiesPath)
		fmt.Println("✓ 现在可以启动 1688-mcp 服务器了")
		return
	}
}

// checkLoginState 检查页面是否已登录
func checkLoginState(page *rod.Page) (bool, error) {
	result, err := page.Eval(`() => {
		// 检查用户头像或昵称元素（1688 登录后的标识）
		var indicators = [
			'[class*="login-name"]',
			'[class*="member-nick"]',
			'[class*="user-name"]',
			'[class*="loginName"]',
			'.header-login-name',
			'[data-biztype="login"]',
		];
		for (var i = 0; i < indicators.length; i++) {
			var el = document.querySelector(indicators[i]);
			if (el && el.textContent.trim().length > 0) return true;
		}
		// 检查页面是否包含登录后特有的字段
		var pageText = document.body ? document.body.innerText : '';
		if (pageText.indexOf('退出登录') >= 0 || pageText.indexOf('我的1688') >= 0) return true;
		return false;
	}`)
	if err != nil {
		return false, err
	}
	return result.Value.Bool(), nil
}
