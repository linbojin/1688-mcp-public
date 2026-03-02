package alibaba

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sirupsen/logrus"
)

// ProductPuhuo 对单个商品 URL 执行"加铺货单"操作（浏览器自动化）
func ProductPuhuo(ctx context.Context, _ *http.Client, rawURL string) (*PuhuoResult, error) {
	logrus.Infof("[puhuo] 铺货: %s", rawURL)

	offerID := extractOfferID(rawURL)
	if offerID == "" {
		return &PuhuoResult{Success: false, Message: "无法从 URL 提取 offer ID"}, nil
	}

	// 1. 加载保存的 cookies
	cdpCookies, err := LoadCDPCookiesRaw()
	if err != nil {
		return &PuhuoResult{Success: false, Message: fmt.Sprintf("加载 cookies 失败: %v", err)}, nil
	}

	// 2. 启动内部浏览器（无头模式）
	l := launcher.New().Headless(true)
	u, err := l.Launch()
	if err != nil {
		return &PuhuoResult{Success: false, Message: fmt.Sprintf("启动浏览器失败: %v", err)}, nil
	}

	browser := rod.New().ControlURL(u).Context(ctx)
	if err := browser.Connect(); err != nil {
		return &PuhuoResult{Success: false, Message: fmt.Sprintf("rod 连接失败: %v", err)}, nil
	}
	defer browser.Close()

	// 3. 打开商品详情页并注入 cookies
	puhuoURL := fmt.Sprintf("https://detail.1688.com/offer/%s.html", offerID)
	page, err := browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return &PuhuoResult{Success: false, Message: fmt.Sprintf("创建页面失败: %v", err)}, nil
	}
	defer page.Close()

	if err := injectCookies(page, cdpCookies); err != nil {
		logrus.Warnf("[puhuo] 注入 cookies 失败: %v", err)
	}

	if err := page.Navigate(puhuoURL); err != nil {
		return &PuhuoResult{Success: false, Message: fmt.Sprintf("打开商品页失败: %v", err)}, nil
	}

	_ = page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width: 1440, Height: 900, DeviceScaleFactor: 1,
	})
	logrus.Infof("[puhuo] 已打开: %s", puhuoURL)

	// 4. 等待商品详情页主体加载
	if err := waitForDetailPage(page); err != nil {
		logrus.Warnf("[puhuo] %v，继续尝试", err)
	}

	// 5. 关闭弹窗
	closePopups(page)

	// 6. 等待"加铺货单"按钮出现
	logrus.Info("[puhuo] 等待加铺货单按钮...")
	if !waitForAddConsignButton(page, 15*time.Second) {
		return &PuhuoResult{
			Success: false,
			Message: "加铺货单按钮未出现，请确认已登录且商品支持代发",
		}, nil
	}
	logrus.Info("[puhuo] 加铺货单按钮已就绪")
	time.Sleep(500 * time.Millisecond)

	// 7. 点击「加铺货单」
	if err := clickAddConsignButton(page); err != nil {
		return &PuhuoResult{Success: false, Message: fmt.Sprintf("点击加铺货单失败: %v", err)}, nil
	}
	logrus.Info("[puhuo] 已点击加铺货单，等待结果...")

	// 8. 等待成功提示
	time.Sleep(3 * time.Second)
	ok, msg := checkPuhuoResult(page)
	return &PuhuoResult{Success: ok, Message: msg}, nil
}

// waitForDetailPage 等待商品详情页主体加载
func waitForDetailPage(page *rod.Page) error {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		result, err := page.Eval(`() => {
	var img = document.querySelector('.detail-gallery-img, .img-view-item img, [class*="gallery"] img');
	var title = document.querySelector('[class*="product-title"], [class*="detail-title"], h1');
	return !!(img || title);
}`)
		if err == nil && result.Value.Bool() {
			return nil
		}
		time.Sleep(800 * time.Millisecond)
	}
	return fmt.Errorf("商品详情页 15s 内未加载完成")
}

// closePopups 关闭页面上的广告/下载APP弹窗
func closePopups(page *rod.Page) {
	page.Eval(`() => {
	var closeSelectors = [
		'[class*="modal"] [class*="close"]',
		'[class*="dialog"] [class*="close"]',
		'[class*="popup"] [class*="close"]',
		'[aria-label="close"]',
		'[aria-label="关闭"]',
	];
	closeSelectors.forEach(function(sel) {
		var el = document.querySelector(sel);
		if (el && el.offsetParent) el.click();
	});
	var closeTexts = ['关闭', '知道了', '取消', '×', '暂不', '以后再说'];
	closeTexts.forEach(function(text) {
		var els = document.querySelectorAll('button, a, span, i');
		for (var i = 0; i < els.length; i++) {
			if (els[i].textContent.trim() === text && els[i].offsetParent) {
				els[i].click();
				break;
			}
		}
	});
}`)
	time.Sleep(500 * time.Millisecond)
}

// waitForAddConsignButton 等待「加铺货单」按钮出现
func waitForAddConsignButton(page *rod.Page, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		result, err := page.Eval(`() => {
	var btn = document.querySelector('button[data-click="ADD_CONSIGN"]');
	return !!(btn && btn.offsetParent !== null);
}`)
		if err == nil && result.Value.Bool() {
			return true
		}
		time.Sleep(600 * time.Millisecond)
	}
	return false
}

// clickAddConsignButton 用 Rod 真实鼠标点击「加铺货单」按钮
func clickAddConsignButton(page *rod.Page) error {
	btn, err := page.Element(`button[data-click="ADD_CONSIGN"]`)
	if err != nil {
		return fmt.Errorf("未找到 button[data-click=ADD_CONSIGN]: %w", err)
	}
	return btn.Click(proto.InputMouseButtonLeft, 1)
}

// checkPuhuoResult 检查铺货是否成功
func checkPuhuoResult(page *rod.Page) (bool, string) {
	result, err := page.Eval(`() => {
	var successTexts = ['铺货成功', '已铺货', '操作成功', '添加成功', '加入铺货单成功'];
	var failTexts = ['铺货失败', '操作失败', '已存在', '重复铺货'];
	var els = document.querySelectorAll('*');
	for (var i = 0; i < els.length; i++) {
		var t = els[i].textContent.trim();
		for (var j = 0; j < successTexts.length; j++) {
			if (t === successTexts[j]) return JSON.stringify({ok: true, msg: t});
		}
		for (var j = 0; j < failTexts.length; j++) {
			if (t.indexOf(failTexts[j]) >= 0) return JSON.stringify({ok: false, msg: t});
		}
	}
	return JSON.stringify({ok: true, msg: '加铺货单操作已提交'});
}`)
	if err != nil {
		return false, fmt.Sprintf("检查结果失败: %v", err)
	}

	var res struct {
		OK  bool   `json:"ok"`
		Msg string `json:"msg"`
	}
	if err := json.Unmarshal([]byte(result.Value.String()), &res); err != nil {
		return true, "加铺货单操作已完成"
	}
	return res.OK, res.Msg
}

// extractOfferID 从 URL 中提取 offer ID
func extractOfferID(rawURL string) string {
	re := regexp.MustCompile(`/offer/(\d+)\.html`)
	if m := re.FindStringSubmatch(rawURL); len(m) > 1 {
		return m[1]
	}
	return ""
}
