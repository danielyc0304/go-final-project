package services

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// PriceCache 價格快取（執行緒安全）
type PriceCache struct {
	mu         sync.RWMutex
	prices     map[string]float64 // symbol -> price
	lastUpdate map[string]time.Time
}

var GlobalPriceCache = &PriceCache{
	prices:     make(map[string]float64),
	lastUpdate: make(map[string]time.Time),
}

// BinanceTradeMessage Binance 交易訊息格式
type BinanceTradeMessage struct {
	Stream string `json:"stream"`
	Data   struct {
		EventType string `json:"e"`
		EventTime int64  `json:"E"`
		Symbol    string `json:"s"`
		Price     string `json:"p"`
		Quantity  string `json:"q"`
	} `json:"data"`
}

// UpdatePrice 更新價格（從 Binance WebSocket 呼叫）
func (pc *PriceCache) UpdatePrice(message []byte) {
	var tradeMsg BinanceTradeMessage
	if err := json.Unmarshal(message, &tradeMsg); err != nil {
		// 訊息格式錯誤，忽略
		return
	}

	if tradeMsg.Data.EventType != "trade" {
		return
	}

	symbol := tradeMsg.Data.Symbol
	priceStr := tradeMsg.Data.Price

	// 將價格字串轉為 float64
	var price float64
	if _, err := fmt.Sscanf(priceStr, "%f", &price); err != nil {
		log.Printf("Failed to parse price for %s: %v", symbol, err)
		return
	}

	pc.mu.Lock()
	pc.prices[symbol] = price
	pc.lastUpdate[symbol] = time.Now()
	pc.mu.Unlock()

	// 可選：記錄價格更新（用於調試）
	// log.Printf("Price updated: %s = %.2f", symbol, price)
}

// GetPrice 取得當前價格
func (pc *PriceCache) GetPrice(symbol string) (float64, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	price, ok := pc.prices[symbol]
	return price, ok
}

// GetPriceWithTimeout 取得價格，若超過指定時間未更新則返回錯誤
func (pc *PriceCache) GetPriceWithTimeout(symbol string, timeout time.Duration) (float64, error) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	price, ok := pc.prices[symbol]
	if !ok {
		return 0, fmt.Errorf("price not available for %s", symbol)
	}

	lastUpdate, ok := pc.lastUpdate[symbol]
	if !ok || time.Since(lastUpdate) > timeout {
		return 0, fmt.Errorf("price data is stale for %s", symbol)
	}

	return price, nil
}

// GetAllPrices 取得所有價格
func (pc *PriceCache) GetAllPrices() map[string]float64 {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	// 建立副本以避免外部修改
	result := make(map[string]float64)
	for k, v := range pc.prices {
		result[k] = v
	}
	return result
}
