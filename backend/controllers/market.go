package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	beego "github.com/beego/beego/v2/server/web"
)

type MarketController struct {
	beego.Controller
}

// GetKLines 處理 GET /v1/market/klines
// 參數：symbol (如 BTCUSDT), interval (如 1m), limit (如 1000)
// @router /klines [get]
func (c *MarketController) GetKLines() {
	symbol := c.GetString("symbol", "BTCUSDT")
	interval := c.GetString("interval", "1m")
	limit := c.GetString("limit", "1000")

	// 1. 呼叫幣安 REST API
	url := fmt.Sprintf("https://api.binance.com/api/v3/klines?symbol=%s&interval=%s&limit=%s", symbol, interval, limit)
	resp, err := http.Get(url)
	if err != nil {
		c.Data["json"] = map[string]string{"error": "Failed to connect to Binance"}
		c.ServeJSON()
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.Data["json"] = map[string]string{"error": "Failed to read response"}
		c.ServeJSON()
		return
	}

	// 2. 解析幣安回傳的原始資料 (Array of Arrays)
	// 格式範例: [[1499040000000, "0.01634790", "0.80000000", ...], ...]
	var rawData [][]interface{}
	if err := json.Unmarshal(body, &rawData); err != nil {
		c.Data["json"] = map[string]string{"error": "Failed to parse data"}
		c.ServeJSON()
		return
	}

	// 3. 轉換格式給前端 (Lightweight Charts 需要 time(秒), open, high, low, close)
	var formattedData []map[string]interface{}

	for _, k := range rawData {
		// 幣安時間是毫秒，轉成秒
		timestampFloat, _ := k[0].(float64)
		timestamp := int64(timestampFloat) / 1000

		item := map[string]interface{}{
			"time":  timestamp,
			"open":  strToFloat(k[1]),
			"high":  strToFloat(k[2]),
			"low":   strToFloat(k[3]),
			"close": strToFloat(k[4]),
		}
		formattedData = append(formattedData, item)
	}

	// 4. 回傳成功資料
	c.Data["json"] = map[string]interface{}{
		"success": true,
		"data":    formattedData,
	}
	c.ServeJSON()
}

// 輔助函式：將 interface{} (實際是 string) 轉為 float64
func strToFloat(v interface{}) float64 {
	s, ok := v.(string)
	if !ok {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
