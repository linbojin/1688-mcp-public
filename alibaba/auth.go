package alibaba

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// CheckLogin 验证当前 cookies 是否有效（能否访问登录保护页面）
func CheckLogin(ctx context.Context, client *http.Client) (*LoginStatus, error) {
	const memberURL = "https://member.1688.com/member/addressList.htm"

	req, err := http.NewRequestWithContext(ctx, "GET", memberURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 检查最终 URL（跟随重定向后）
	finalURL := resp.Request.URL.String()
	logrus.Debugf("check_login 最终 URL: %s", finalURL)

	if strings.Contains(finalURL, "login") || strings.Contains(finalURL, "signin") {
		return &LoginStatus{LoggedIn: false}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	html := string(body)

	// 再次检查：若响应体提示未登录
	if strings.Contains(html, "login.taobao.com") || strings.Contains(html, "请登录") {
		return &LoginStatus{LoggedIn: false}, nil
	}

	// 提取用户名
	username := extractUsername(html)
	logrus.Infof("check_login: 已登录, username=%s", username)
	return &LoginStatus{LoggedIn: true, Username: username}, nil
}

// extractUsername 从页面 HTML 中提取用户名
func extractUsername(html string) string {
	for _, pattern := range []string{`"memberNick":"`, `"loginId":"`, `"nick":"`} {
		idx := strings.Index(html, pattern)
		if idx == -1 {
			continue
		}
		start := idx + len(pattern)
		end := strings.Index(html[start:], `"`)
		if end > 0 {
			return html[start : start+end]
		}
	}
	return ""
}
