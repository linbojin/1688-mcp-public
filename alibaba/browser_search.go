package alibaba

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sirupsen/logrus"
)

const searchIndexURL = "https://air.1688.com/app/channel-fe/search/index.html#/"

// SearchViaBrowser 使用 rod 浏览器自动化搜索（内部管理的无头浏览器，注入已保存 cookies）
func SearchViaBrowser(ctx context.Context, keyword string, count int) ([]Product, error) {
	// 1. 加载保存的 cookies
	cdpCookies, err := LoadCDPCookiesRaw()
	if err != nil {
		return nil, fmt.Errorf("加载 cookies 失败: %w", err)
	}

	// 2. 启动内部浏览器（无头模式）
	l := launcher.New().Headless(true)
	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("启动浏览器失败: %w", err)
	}

	browser := rod.New().ControlURL(u).Context(ctx)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("rod 连接失败: %w", err)
	}
	defer browser.Close()
	logrus.Info("[search] 浏览器已启动")

	// 3. 打开搜索入口页并注入 cookies
	searchPage, err := browser.Page(proto.TargetCreateTarget{URL: searchIndexURL})
	if err != nil {
		return nil, fmt.Errorf("打开搜索入口页失败: %w", err)
	}
	defer searchPage.Close()

	if err := injectCookies(searchPage, cdpCookies); err != nil {
		logrus.Warnf("[search] 注入 cookies 失败: %v", err)
	}

	// 重新加载页面使 cookies 生效
	if err := searchPage.Navigate(searchIndexURL); err != nil {
		return nil, fmt.Errorf("导航搜索入口页失败: %w", err)
	}

	if err := searchPage.WaitLoad(); err != nil {
		return nil, fmt.Errorf("等待搜索页加载失败: %w", err)
	}
	time.Sleep(1500 * time.Millisecond)

	// 4. 找到搜索框，输入关键词
	inputEl, err := findSearchInput(searchPage)
	if err != nil {
		return nil, fmt.Errorf("找不到搜索框: %w", err)
	}
	if err := inputEl.Input(keyword); err != nil {
		return nil, fmt.Errorf("输入关键词失败: %w", err)
	}

	// 5. 记录当前 tab 列表，准备检测新 tab
	pagesBefore, _ := browser.Pages()

	// 按 Enter 提交搜索
	if err := inputEl.Type(input.Enter); err != nil {
		if err2 := clickSearchButton(searchPage); err2 != nil {
			return nil, fmt.Errorf("提交搜索失败: %v / %v", err, err2)
		}
	}

	// 6. 等待结果页（新 tab）打开
	resultPage, err := waitForNewPage(browser, pagesBefore, 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("等待搜索结果页失败: %w", err)
	}
	defer resultPage.Close()

	logrus.Infof("[search] 结果页 URL: %s", resultPage.MustInfo().URL)

	// 7. 等待商品列表渲染完成
	if err := waitForProductsRendered(resultPage); err != nil {
		logrus.Warnf("[search] 商品列表等待超时，继续尝试: %v", err)
	}

	// 8. 依次点击 4 个过滤条件
	applyDropshipFilters(resultPage)

	// 等待过滤后重新渲染
	time.Sleep(2 * time.Second)

	// 9. 提取商品数据
	products, err := extractProducts(resultPage)
	if err != nil {
		return nil, fmt.Errorf("提取商品失败: %w", err)
	}
	logrus.Infof("[search] 提取到 %d 个商品，取前 %d", len(products), count)

	if len(products) > count {
		products = products[:count]
	}
	return products, nil
}

// injectCookies 将已保存的 cookies 注入到页面
func injectCookies(page *rod.Page, cdpCookies []CDPCookie) error {
	params := make([]*proto.NetworkCookieParam, 0, len(cdpCookies))
	for _, c := range cdpCookies {
		params = append(params, &proto.NetworkCookieParam{
			Name:   c.Name,
			Value:  c.Value,
			Domain: c.Domain,
			Path:   c.Path,
		})
	}
	return page.SetCookies(params)
}

// findSearchInput 找到搜索输入框
func findSearchInput(page *rod.Page) (*rod.Element, error) {
	selectors := []string{
		`input[type="search"]`,
		`input[placeholder*="搜索"]`,
		`input[class*="search"]`,
		`input[class*="Search"]`,
		`.search-input input`,
		`input`,
	}
	for _, sel := range selectors {
		el, err := page.Element(sel)
		if err == nil && el != nil {
			return el, nil
		}
	}
	return nil, fmt.Errorf("所有选择器均未找到搜索框")
}

// clickSearchButton 点击搜索按钮（Enter 失败时备用）
func clickSearchButton(page *rod.Page) error {
	selectors := []string{
		`button[type="submit"]`,
		`[class*="search-btn"]`,
		`[class*="searchBtn"]`,
		`button[class*="search"]`,
	}
	for _, sel := range selectors {
		el, err := page.Element(sel)
		if err == nil && el != nil {
			return el.Click(proto.InputMouseButtonLeft, 1)
		}
	}
	return fmt.Errorf("未找到搜索按钮")
}

// waitForNewPage 等待新 tab 打开并返回
func waitForNewPage(browser *rod.Browser, before rod.Pages, timeout time.Duration) (*rod.Page, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		after, err := browser.Pages()
		if err != nil {
			continue
		}
		for _, p := range after {
			isNew := true
			for _, old := range before {
				if p.TargetID == old.TargetID {
					isNew = false
					break
				}
			}
			if isNew {
				if err := p.WaitLoad(); err != nil {
					logrus.Warnf("[search] 新 tab 加载等待失败: %v", err)
				}
				time.Sleep(2 * time.Second)
				return p, nil
			}
		}
	}
	return nil, fmt.Errorf("超时 %v 未检测到新 tab", timeout)
}

// waitForProductsRendered 等待商品列表 JS 渲染完成
func waitForProductsRendered(page *rod.Page) error {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		result, err := page.Eval(`
() => {
	var s = window.__INITIAL_STATE__;
	if (s) {
		var list = (s.offerList) || (s.data && s.data.offerList) || (s.data && s.data.resultList) || [];
		if (list.length > 0) return list.length;
	}
	var cards = document.querySelectorAll('[class*="offer"],[class*="card"],[class*="product"]');
	return cards.length;
}
`)
		if err == nil {
			n := result.Value.Int()
			if n > 0 {
				logrus.Infof("[search] 商品列表已渲染，数量=%d", n)
				return nil
			}
		}
		time.Sleep(800 * time.Millisecond)
	}
	return fmt.Errorf("商品列表未在 15s 内渲染")
}

// applyDropshipFilters 依次点击 4 个过滤条件
func applyDropshipFilters(page *rod.Page) {
	const filterWait = 5 * time.Second

	// 1. 包邮
	if ok := clickFilterText(page, "包邮"); ok {
		logrus.Info("[search] 已点击过滤条件: 包邮")
		time.Sleep(filterWait)
	} else {
		logrus.Warn("[search] 未找到过滤条件: 包邮")
	}

	// 2. 一件代发
	if ok := clickFilterText(page, "一件代发"); ok {
		logrus.Info("[search] 已点击过滤条件: 一件代发")
		time.Sleep(filterWait)
	} else {
		logrus.Warn("[search] 未找到过滤条件: 一件代发")
	}

	// 3. 急速发货（air.1688.com 发货时效过滤，对应 48H 发货）
	clickFilterText(page, "发货时效")
	time.Sleep(800 * time.Millisecond)
	if ok := clickFilterText(page, "急速发货"); ok {
		logrus.Info("[search] 已点击过滤条件: 急速发货")
		time.Sleep(filterWait)
	} else {
		logrus.Warn("[search] 未找到过滤条件: 急速发货")
	}

	// 4. 抖音密文面单
	clickFilterText(page, "更多")
	time.Sleep(800 * time.Millisecond)
	if ok := clickFilterText(page, "抖音"); ok {
		logrus.Info("[search] 已点击过滤条件: 抖音")
		time.Sleep(filterWait)
	} else {
		logrus.Warn("[search] 未找到过滤条件: 抖音")
	}
}

// clickFilterText 通过文本内容找到过滤项并点击
func clickFilterText(page *rod.Page, text string) bool {
	result, err := page.Eval(
		`(text) => {
			var selectors = [
				'[class*="filter"] span',
				'[class*="filter"] label',
				'[class*="screen"] span',
				'[class*="condition"] span',
				'[class*="tag"] span',
				'[class*="check"] label',
				'[class*="item"] span',
				'button',
				'label',
				'span',
			];
			for (var i = 0; i < selectors.length; i++) {
				var els = document.querySelectorAll(selectors[i]);
				for (var j = 0; j < els.length; j++) {
					var el = els[j];
					if (el.textContent.trim() === text && el.offsetParent !== null) {
						el.click();
						return true;
					}
				}
			}
			return false;
		}`,
		text,
	)
	if err != nil {
		return false
	}
	return result.Value.Bool()
}

// extractProducts 从结果页提取商品列表
func extractProducts(page *rod.Page) ([]Product, error) {
	// 先尝试从 __INITIAL_STATE__ 提取
	stateProducts, err := extractFromStateJS(page)
	if err == nil && len(stateProducts) > 0 {
		logrus.Infof("[search] 从 __INITIAL_STATE__ 提取到 %d 个商品", len(stateProducts))
		return stateProducts, nil
	}

	// 降级：DOM 提取
	domProducts, err := extractFromDOMJS(page)
	if err != nil {
		dumpPageDebug(page)
		return nil, err
	}
	logrus.Infof("[search] 从 DOM 提取到 %d 个商品", len(domProducts))
	return domProducts, nil
}

// extractFromStateJS 从 window.__INITIAL_STATE__ 提取商品
func extractFromStateJS(page *rod.Page) ([]Product, error) {
	result, err := page.Eval(`
() => {
	var s = window.__INITIAL_STATE__;
	if (!s) return null;
	var list = s.offerList
		|| (s.data && s.data.offerList)
		|| (s.data && s.data.resultList)
		|| s.resultList
		|| [];
	if (!list.length) return null;
	return JSON.stringify(list);
}
`)
	if err != nil {
		return nil, err
	}
	if result.Value.Nil() {
		return nil, fmt.Errorf("__INITIAL_STATE__ 无数据")
	}

	var items []map[string]interface{}
	if err := json.Unmarshal([]byte(result.Value.String()), &items); err != nil {
		return nil, err
	}
	return parseProductsFromState(items), nil
}

// extractFromDOMJS 从 DOM 提取商品（包含商卡全字段）
func extractFromDOMJS(page *rod.Page) ([]Product, error) {
	result, err := page.Eval(`
() => {
	// 找商品卡片列表
	var cardSelectors = [
		'[class*="offer-card"]',
		'[class*="offer-item"]',
		'[class*="card-item"]',
		'[class*="product-item"]',
		'[data-offer-id]',
	];
	var cards = [];
	for (var i = 0; i < cardSelectors.length; i++) {
		var found = document.querySelectorAll(cardSelectors[i]);
		if (found.length > 0) { cards = Array.from(found); break; }
	}
	if (!cards.length) return null;

	function matchLeaf(card, pattern) {
		var els = card.querySelectorAll('*');
		for (var i = 0; i < els.length; i++) {
			var el = els[i];
			if (el.children.length > 0) continue;
			var t = el.textContent.trim();
			var m = t.match(pattern);
			if (m) return m;
		}
		return null;
	}

	function hasText(card, str) {
		return card.textContent.indexOf(str) >= 0;
	}

	var products = [];
	for (var i = 0; i < cards.length; i++) {
		var card = cards[i];

		var detailUrl = '';
		var offerId = '';
		if (card.tagName === 'A' && card.href) {
			detailUrl = card.href;
		} else {
			var links = card.querySelectorAll('a[href*="detail.1688.com"]');
			if (links.length) detailUrl = links[0].href;
		}
		if (detailUrl) {
			var m = detailUrl.match(/\/offer\/(\d+)\.html/);
			if (m) offerId = m[1];
		}
		if (!offerId) {
			var reportEl = card.querySelector('[data-aplus-report]') || card;
			var report = reportEl.getAttribute('data-aplus-report') || '';
			var rm = report.match(/object_id@(\d+)/);
			if (rm) offerId = rm[1];
		}
		if (!offerId) offerId = card.getAttribute('data-offer-id') || '';

		var titleEl = card.querySelector('[class*="title"],[class*="subject"],h2,h3');
		var title = titleEl ? titleEl.textContent.trim() : '';

		var priceEl = card.querySelector('[class*="price"]');
		var priceText = priceEl ? priceEl.textContent.replace(/,/g, '').trim() : '';
		var priceM = priceText.match(/[\d.]+/);
		var price = priceM ? parseFloat(priceM[0]) : 0;

		var companyEl = card.querySelector('[class*="company"],[class*="supplier"],[class*="seller"],[class*="shop"]');
		var supplier = companyEl ? companyEl.textContent.trim() : '';
		supplier = supplier.replace(/^\d+年\s*/, '');

		var yearEl = card.querySelector('[class*="year"],[class*="integrity"],[class*="credit"]');
		var yearText = yearEl ? yearEl.textContent.trim() : '';
		var yearM = yearText.match(/(\d+)/);
		var years = yearM ? parseInt(yearM[1]) : 0;

		var imgEl = card.querySelector('img');
		var imageUrl = imgEl ? (imgEl.getAttribute('data-src') || imgEl.src || '') : '';
		if (imageUrl.indexOf('//') === 0) imageUrl = 'https:' + imageUrl;

		var freeShipping = hasText(card, '包邮') || hasText(card, '免邮');
		var oneDropship = hasText(card, '一件代发') || hasText(card, '代发');

		var monthlyDropship = '';
		var weekly7Dropship = '';
		var countEls = Array.from(card.querySelectorAll('p[class*="offer-body__count"],p[class*="count"]'));
		for (var ci = 0; ci < countEls.length; ci++) {
			var pel = countEls[ci];
			var label = '';
			for (var ni = 0; ni < pel.childNodes.length; ni++) {
				var node = pel.childNodes[ni];
				if (node.nodeType === 3 && node.textContent.trim()) {
					label = node.textContent.trim();
					break;
				}
			}
			var valSpan = pel.querySelector('span');
			var val = valSpan ? valSpan.textContent.trim() : '';
			if (label === '月代发' && val) monthlyDropship = val;
			if (label === '近7天代发' && val) weekly7Dropship = val;
		}

		var pickupRate48h = 0;
		var r48m = matchLeaf(card, /48H支揽率\s*([\d.]+)%/);
		if (r48m) pickupRate48h = parseFloat(r48m[1]);

		var pickupRate24h = 0;
		var returnRate = 0;
		var bareRates = [];
		var allLeaves = Array.from(card.querySelectorAll('*'))
			.filter(function(e){ return e.children.length === 0; })
			.map(function(e){ return e.textContent.trim(); });
		for (var li = 0; li < allLeaves.length; li++) {
			var t = allLeaves[li];
			if (/48H支揽率/.test(t)) continue;
			var bm = t.match(/^([\d.]+)%$/);
			if (bm) bareRates.push(parseFloat(bm[1]));
		}
		if (bareRates.length >= 1) pickupRate24h = bareRates[0];
		if (bareRates.length >= 2) returnRate = bareRates[1];

		var monthlySales = 0;
		var salesEl = card.querySelector('[class*="sale"],[class*="sold"],[class*="month-sale"]');
		if (salesEl) {
			var sm = salesEl.textContent.match(/(\d+)/);
			if (sm) monthlySales = parseInt(sm[1]);
		}

		var secretPlatforms = [];
		var secretSection = null;
		var allEls = Array.from(card.querySelectorAll('*'));
		for (var si = 0; si < allEls.length; si++) {
			if (allEls[si].textContent.trim() === '密文面单') {
				secretSection = allEls[si].parentElement || allEls[si];
				break;
			}
		}
		if (secretSection) {
			var icons = Array.from(secretSection.querySelectorAll('img,i,[class*="icon"],[class*="logo"],[class*="platform"]'));
			icons.forEach(function(icon) {
				var info = ((icon.className || '') + ' ' + (icon.src || '') + ' ' + (icon.alt || '') + ' ' + icon.textContent).toLowerCase();
				if (/taobao|淘宝/.test(info) && secretPlatforms.indexOf('淘宝') < 0) secretPlatforms.push('淘宝');
				if (/douyin|抖音|tiktok/.test(info) && secretPlatforms.indexOf('抖音') < 0) secretPlatforms.push('抖音');
				if (/xhs|xiaohongshu|小红书|rednote/.test(info) && secretPlatforms.indexOf('小红书') < 0) secretPlatforms.push('小红书');
				if (/pinduoduo|pdd|拼多多/.test(info) && secretPlatforms.indexOf('拼多多') < 0) secretPlatforms.push('拼多多');
			});
			var st = secretSection.textContent;
			if (st.indexOf('淘宝') >= 0 && secretPlatforms.indexOf('淘宝') < 0) secretPlatforms.push('淘宝');
			if (st.indexOf('抖音') >= 0 && secretPlatforms.indexOf('抖音') < 0) secretPlatforms.push('抖音');
			if (st.indexOf('小红书') >= 0 && secretPlatforms.indexOf('小红书') < 0) secretPlatforms.push('小红书');
		}
		var supportsDouyin = secretPlatforms.indexOf('抖音') >= 0 || hasText(card, '抖音');

		var taobaoScore = '';
		var tbsm = matchLeaf(card, /淘宝商品体验分\s*(.+)/);
		if (tbsm) taobaoScore = tbsm[1].trim();

		var logisticsScore = '';
		var lgsm = matchLeaf(card, /下游物流分[:：]?\s*(.+)/);
		if (lgsm) logisticsScore = lgsm[1].trim();

		var listingCount = '';
		var lcm = matchLeaf(card, /铺货数[:：\s]*(.+)/);
		if (lcm) listingCount = lcm[1].trim();

		if (offerId || title) {
			products.push({
				offer_id: offerId,
				title: title,
				price: price,
				supplier: supplier,
				years: years,
				detail_url: offerId ? 'https://detail.1688.com/offer/' + offerId + '.html' : detailUrl,
				image_url: imageUrl,
				free_shipping: freeShipping,
				one_dropship: oneDropship,
				monthly_dropship: monthlyDropship,
				weekly7_dropship: weekly7Dropship,
				pickup_rate_48h: pickupRate48h,
				pickup_rate_24h: pickupRate24h,
				monthly_sales: monthlySales,
				supports_douyin: supportsDouyin,
				secret_waybill_platforms: secretPlatforms,
				taobao_score: taobaoScore,
				logistics_score: logisticsScore,
				return_rate: returnRate,
				listing_count: listingCount,
			});
		}
	}
	return products.length ? JSON.stringify(products) : null;
}
`)
	if err != nil {
		return nil, fmt.Errorf("DOM 提取 JS 失败: %w", err)
	}
	if result.Value.Nil() {
		return nil, fmt.Errorf("DOM 未找到商品卡片")
	}

	var items []map[string]interface{}
	if err := json.Unmarshal([]byte(result.Value.String()), &items); err != nil {
		return nil, err
	}
	return parseProductsFromDOM(items), nil
}

// dumpPageDebug 输出页面调试信息
func dumpPageDebug(page *rod.Page) {
	result, err := page.Eval(`
() => {
	return JSON.stringify({
		url: location.href,
		title: document.title,
		bodyLen: document.body ? document.body.innerHTML.length : 0,
		hasState: !!window.__INITIAL_STATE__,
		topClasses: Array.from(document.querySelectorAll('[class]'))
			.slice(0, 20)
			.map(el => el.className.toString().split(' ')[0])
			.filter(Boolean),
	});
}
`)
	if err != nil {
		logrus.Errorf("[search] debug dump 失败: %v", err)
		return
	}
	logrus.Infof("[search] 页面调试信息: %s", result.Value.String())
}

// parseProductsFromState 从 __INITIAL_STATE__ 数据解析商品
func parseProductsFromState(items []map[string]interface{}) []Product {
	var products []Product
	for _, item := range items {
		p := Product{
			OfferID:        jsonStr(item, "offerId", "id"),
			Title:          jsonStr(item, "title", "subject"),
			Price:          jsonFloat(item, "price"),
			Supplier:       jsonStrNested(item, []string{"company", "name"}, "sellerName"),
			Years:          int(jsonFloat(item, "company.years", "sellerYears")),
			ImageURL:       jsonStr(item, "imageUrl"),
			FreeShipping:   jsonBool(item, "freePostFee", "freeShipping"),
			OneDropship:    jsonBool(item, "consignType", "singleFilter"),
			SupportsDouyin: jsonBool(item, "secretWaybill", "supportDouyin"),
		}
		if p.OfferID != "" {
			p.DetailURL = "https://detail.1688.com/offer/" + p.OfferID + ".html"
		}

		switch v := item["monthlySales"].(type) {
		case float64:
			p.MonthlySales = int(v)
		case string:
			fmt.Sscanf(strings.TrimSpace(v), "%d", &p.MonthlySales)
		}

		switch v := item["delivery48h"].(type) {
		case float64:
			p.PickupRate48h = v
		case string:
			s := strings.TrimSuffix(strings.TrimSpace(v), "%")
			fmt.Sscanf(s, "%f", &p.PickupRate48h)
		}

		products = append(products, p)
	}
	return products
}

// parseProductsFromDOM 从 DOM 提取的 map 解析商品
func parseProductsFromDOM(items []map[string]interface{}) []Product {
	var products []Product
	for _, item := range items {
		p := Product{
			OfferID:         jsonStr(item, "offer_id"),
			Title:           jsonStr(item, "title"),
			Price:           jsonFloat(item, "price"),
			Supplier:        jsonStr(item, "supplier"),
			DetailURL:       jsonStr(item, "detail_url"),
			ImageURL:        jsonStr(item, "image_url"),
			FreeShipping:    jsonBool(item, "free_shipping"),
			OneDropship:     jsonBool(item, "one_dropship"),
			MonthlyDropship: jsonStr(item, "monthly_dropship"),
			Weekly7Dropship: jsonStr(item, "weekly7_dropship"),
			PickupRate48h:   jsonFloat(item, "pickup_rate_48h"),
			PickupRate24h:   jsonFloat(item, "pickup_rate_24h"),
			SupportsDouyin:  jsonBool(item, "supports_douyin"),
			TaobaoScore:     jsonStr(item, "taobao_score"),
			LogisticsScore:  jsonStr(item, "logistics_score"),
			ReturnRate:      jsonFloat(item, "return_rate"),
			ListingCount:    jsonStr(item, "listing_count"),
		}
		p.Years = int(jsonFloat(item, "years"))
		p.MonthlySales = int(jsonFloat(item, "monthly_sales"))

		if raw, ok := item["secret_waybill_platforms"]; ok {
			if arr, ok := raw.([]interface{}); ok {
				for _, v := range arr {
					if s, ok := v.(string); ok {
						p.SecretWaybillPlatforms = append(p.SecretWaybillPlatforms, s)
					}
				}
			}
		}

		products = append(products, p)
	}
	return products
}
