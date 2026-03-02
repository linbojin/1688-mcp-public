package alibaba

// Product 搜索结果中的商品
type Product struct {
	OfferID  string  `json:"offer_id"`
	Title    string  `json:"title"`
	Price    float64 `json:"price"`
	Supplier string  `json:"supplier"`
	Years    int     `json:"years"`

	// 链接 / 图片
	DetailURL string `json:"detail_url"`
	ImageURL  string `json:"image_url,omitempty"`

	// 发货 & 代发
	FreeShipping    bool   `json:"free_shipping,omitempty"`
	OneDropship     bool   `json:"one_dropship,omitempty"`
	MonthlyDropship string `json:"monthly_dropship,omitempty"` // 月代发量，如 "100+"
	Weekly7Dropship string `json:"weekly7_dropship,omitempty"` // 近7天代发量，如 "100以内"

	// 发货率
	PickupRate48h float64 `json:"pickup_rate_48h,omitempty"` // 48H支揽率 (%)
	PickupRate24h float64 `json:"pickup_rate_24h,omitempty"` // 24H支揽率 (%)

	// 销量
	MonthlySales int `json:"monthly_sales,omitempty"` // 月销量

	// 平台 & 密文面单
	SupportsDouyin         bool     `json:"supports_douyin,omitempty"`
	SecretWaybillPlatforms []string `json:"secret_waybill_platforms,omitempty"` // 如 ["淘宝","抖音","小红书"]

	// 质量评分
	TaobaoScore    string  `json:"taobao_score,omitempty"`    // 淘宝商品体验分，如 "高"
	LogisticsScore string  `json:"logistics_score,omitempty"` // 下游物流分，如 "高"
	ReturnRate     float64 `json:"return_rate,omitempty"`     // 商品退货率 (%)

	// 铺货信息
	ListingCount string `json:"listing_count,omitempty"` // 铺货数，如 "200+"
}

// PuhuoResult 铺货结果
type PuhuoResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// LoginStatus 登录状态
type LoginStatus struct {
	LoggedIn bool   `json:"logged_in"`
	Username string `json:"username,omitempty"`
}

// RefreshResult cookies 刷新结果
type RefreshResult struct {
	Success     bool   `json:"success"`
	CookieCount int    `json:"cookie_count"`
	Message     string `json:"message"`
}
