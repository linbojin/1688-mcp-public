package alibaba

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"

	appErrors "github.com/linbojin/1688-mcp-public/errors"
)

// Search1688 搜索 1688 商品
// 优先使用浏览器自动化（内部无头浏览器），失败时降级到纯 HTTP
// count 为最终返回条数（默认 20）
func Search1688(ctx context.Context, client *http.Client, keyword string, count int) ([]Product, error) {
	if count <= 0 {
		count = 20
	}
	if count > 20 {
		count = 20
	}
	logrus.Infof("搜索 1688 商品: keyword=%s, count=%d", keyword, count)

	// 优先：浏览器自动化（真实点击过滤条件，字段完整）
	products, err := SearchViaBrowser(ctx, keyword, count)
	if err == nil && len(products) > 0 {
		logrus.Infof("搜索完成（浏览器），共 %d 条结果", len(products))
		return products, nil
	}
	logrus.Warnf("浏览器搜索失败: %v，降级到纯 HTTP", err)

	// 降级：纯 HTTP（字段可能不完整，过滤靠 URL 参数）
	const buffer = 50
	products, err = searchViaAirHTML(ctx, client, keyword, buffer)
	if err == nil && len(products) > 0 {
		result := filterDropshipProducts(products, count)
		logrus.Infof("搜索完成（air HTML 降级），共 %d 条结果", len(result))
		return result, nil
	}

	products, err = searchViaSHTML(ctx, client, keyword, buffer)
	if err == nil && len(products) > 0 {
		result := filterDropshipProducts(products, count)
		logrus.Infof("搜索完成（s.1688.com 降级），共 %d 条结果", len(result))
		return result, nil
	}

	return nil, appErrors.ErrSearchNoResult
}

// searchViaAirHTML 请求 air.1688.com 搜索结果页，从 HTML 中提取 __INITIAL_STATE__
func searchViaAirHTML(ctx context.Context, client *http.Client, keyword string, limit int) ([]Product, error) {
	searchURL := fmt.Sprintf(
		"https://air.1688.com/app/channel-fe/search/result.html?keywords=%s&n=y&consignType=1&freePostFee=true",
		url.QueryEscape(keyword),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	setSearchHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 air.1688.com 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	html := string(body)

	return extractProductsFromHTML(html, limit)
}

// searchViaSHTML 请求 s.1688.com 搜索，解析 HTML 中的商品数据
func searchViaSHTML(ctx context.Context, client *http.Client, keyword string, limit int) ([]Product, error) {
	searchURL := fmt.Sprintf(
		"https://s.1688.com/selloffer/offer_search.htm?keywords=%s&n=y&consignType=1&freePostFee=true",
		url.QueryEscape(keyword),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	setSearchHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 s.1688.com 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	html := string(body)

	return extractProductsFromHTML(html, limit)
}

// extractProductsFromHTML 从 HTML 中提取商品列表（多种策略）
func extractProductsFromHTML(html string, limit int) ([]Product, error) {
	// 策略1: 从 window.__INITIAL_STATE__ 提取 JSON
	products, err := extractFromInitialState(html, limit)
	if err == nil && len(products) > 0 {
		return products, nil
	}

	// 策略2: 从 JSON 数据块中提取（页面内嵌 JSON）
	products, err = extractFromEmbeddedJSON(html, limit)
	if err == nil && len(products) > 0 {
		return products, nil
	}

	// 策略3: 正则提取 offer 链接和基本信息
	products = extractFromLinks(html, limit)
	if len(products) > 0 {
		return products, nil
	}

	return nil, fmt.Errorf("无法从 HTML 中提取商品数据")
}

// extractFromInitialState 从 __INITIAL_STATE__ 全局变量提取
func extractFromInitialState(html string, limit int) ([]Product, error) {
	re := regexp.MustCompile(`window\.__INITIAL_STATE__\s*=\s*(\{[\s\S]*?\})\s*;?\s*(?:</script>|window\.)`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		return nil, fmt.Errorf("未找到 __INITIAL_STATE__")
	}

	var state map[string]interface{}
	if err := json.Unmarshal([]byte(matches[1]), &state); err != nil {
		return nil, err
	}

	var items []interface{}
	for _, path := range [][]string{
		{"offerList"},
		{"data", "offerList"},
		{"resultList"},
		{"data", "resultList"},
		{"mainInfo", "offerList"},
	} {
		obj := interface{}(state)
		for _, key := range path {
			if m, ok := obj.(map[string]interface{}); ok {
				obj = m[key]
			} else {
				obj = nil
				break
			}
		}
		if arr, ok := obj.([]interface{}); ok && len(arr) > 0 {
			items = arr
			break
		}
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("__INITIAL_STATE__ 中无商品列表")
	}

	var products []Product
	for i, raw := range items {
		if i >= limit {
			break
		}
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		monthlySalesStr := jsonStr(item, "monthlySales", "quantitySumMonth")
		monthlySalesInt := 0
		fmt.Sscanf(monthlySalesStr, "%d", &monthlySalesInt)
		p := Product{
			OfferID:        jsonStr(item, "offerId", "id"),
			Title:          jsonStr(item, "title", "subject"),
			Price:          jsonFloat(item, "price"),
			Supplier:       jsonStrNested(item, []string{"company", "name"}, "sellerName"),
			Years:          int(jsonFloat(item, "company.years", "sellerYears")),
			MonthlySales:   monthlySalesInt,
			ImageURL:       jsonStr(item, "imageUrl"),
			FreeShipping:   jsonBool(item, "freePostFee", "freightFee", "freeShipping"),
			OneDropship:    jsonBool(item, "consignType", "tradeType", "singleFilter"),
			SupportsDouyin: jsonBool(item, "secretWaybill", "supportDouyin"),
		}
		if p.OfferID != "" {
			p.DetailURL = "https://detail.1688.com/offer/" + p.OfferID + ".html"
		}
		if p.DetailURL == "" {
			p.DetailURL = jsonStr(item, "detailUrl")
		}
		products = append(products, p)
	}

	return products, nil
}

// extractFromEmbeddedJSON 从页面内嵌 JSON 数据中提取
func extractFromEmbeddedJSON(html string, limit int) ([]Product, error) {
	re := regexp.MustCompile(`"offerId"\s*:\s*"(\d+)"`)
	allMatches := re.FindAllStringSubmatch(html, -1)
	if len(allMatches) == 0 {
		return nil, fmt.Errorf("未找到嵌入的 offerId")
	}

	seen := make(map[string]bool)
	var products []Product
	for _, m := range allMatches {
		if len(products) >= limit {
			break
		}
		offerID := m[1]
		if seen[offerID] {
			continue
		}
		seen[offerID] = true

		products = append(products, Product{
			OfferID:   offerID,
			DetailURL: "https://detail.1688.com/offer/" + offerID + ".html",
		})
	}

	return products, nil
}

// extractFromLinks 从 HTML 链接中提取 offer ID
func extractFromLinks(html string, limit int) []Product {
	re := regexp.MustCompile(`detail\.1688\.com/offer/(\d+)\.html`)
	allMatches := re.FindAllStringSubmatch(html, -1)

	seen := make(map[string]bool)
	var products []Product
	for _, m := range allMatches {
		if len(products) >= limit {
			break
		}
		offerID := m[1]
		if seen[offerID] {
			continue
		}
		seen[offerID] = true

		products = append(products, Product{
			OfferID:   offerID,
			DetailURL: "https://detail.1688.com/offer/" + offerID + ".html",
		})
	}

	return products
}

// filterDropshipProducts 后过滤：尝试筛选代发友好商品，取前 count 条
func filterDropshipProducts(products []Product, count int) []Product {
	hasDropshipData := false
	for _, p := range products {
		if p.OneDropship || p.FreeShipping {
			hasDropshipData = true
			break
		}
	}

	if !hasDropshipData {
		if len(products) > count {
			return products[:count]
		}
		return products
	}

	var filtered []Product
	for _, p := range products {
		if len(filtered) >= count {
			break
		}
		if p.OneDropship || p.FreeShipping {
			filtered = append(filtered, p)
		}
	}

	if len(filtered) < count {
		for _, p := range products {
			if len(filtered) >= count {
				break
			}
			if !p.OneDropship && !p.FreeShipping {
				filtered = append(filtered, p)
			}
		}
	}

	return filtered
}

func setSearchHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://air.1688.com/")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
}

// --- JSON 辅助函数 ---

func jsonStr(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if strings.Contains(key, ".") {
			parts := strings.SplitN(key, ".", 2)
			if sub, ok := m[parts[0]].(map[string]interface{}); ok {
				if v, ok := sub[parts[1]]; ok {
					return fmt.Sprintf("%v", v)
				}
			}
			continue
		}
		if v, ok := m[key]; ok && v != nil {
			switch val := v.(type) {
			case string:
				return val
			case float64:
				return strconv.FormatFloat(val, 'f', -1, 64)
			default:
				return fmt.Sprintf("%v", val)
			}
		}
	}
	return ""
}

func jsonFloat(m map[string]interface{}, keys ...string) float64 {
	for _, key := range keys {
		if strings.Contains(key, ".") {
			parts := strings.SplitN(key, ".", 2)
			if sub, ok := m[parts[0]].(map[string]interface{}); ok {
				if v, ok := sub[parts[1]]; ok {
					if f, ok := v.(float64); ok {
						return f
					}
				}
			}
			continue
		}
		if v, ok := m[key]; ok {
			switch val := v.(type) {
			case float64:
				return val
			case string:
				f, _ := strconv.ParseFloat(val, 64)
				return f
			}
		}
	}
	return 0
}

func jsonStrNested(m map[string]interface{}, nestedPath []string, fallbackKeys ...string) string {
	if len(nestedPath) >= 2 {
		if sub, ok := m[nestedPath[0]].(map[string]interface{}); ok {
			if v, ok := sub[nestedPath[1]]; ok {
				return fmt.Sprintf("%v", v)
			}
		}
	}
	return jsonStr(m, fallbackKeys...)
}

func jsonBool(m map[string]interface{}, keys ...string) bool {
	for _, key := range keys {
		v, ok := m[key]
		if !ok || v == nil {
			continue
		}
		switch val := v.(type) {
		case bool:
			return val
		case float64:
			return val != 0
		case string:
			low := strings.ToLower(val)
			return low == "true" || low == "1" || low == "yes"
		}
	}
	return false
}
