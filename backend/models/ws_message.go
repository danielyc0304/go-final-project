package models

import (
	"encoding/json"
	"time"
)

// WSMessageType WebSocket 消息類型
type WSMessageType string

const (
	WSMessageTypeOrderExecuted          WSMessageType = "ORDER_EXECUTED"           // 訂單成交
	WSMessageTypeLimitOrderFilled       WSMessageType = "LIMIT_ORDER_FILLED"       // 限價單成交
	WSMessageTypeLeveragePositionOpened WSMessageType = "LEVERAGE_POSITION_OPENED" // 槓桿位置開倉
	WSMessageTypeLeveragePositionClosed WSMessageType = "LEVERAGE_POSITION_CLOSED" // 槓桿位置平倉
	WSMessageTypeError                  WSMessageType = "ERROR"                    // 錯誤
)

// WSMessage WebSocket 消息基礎結構
type WSMessage struct {
	Type      WSMessageType `json:"type"`      // 消息類型
	Timestamp time.Time     `json:"timestamp"` // 時間戳
	Data      interface{}   `json:"data"`      // 數據
}

// OrderExecutedData 訂單成交數據
type OrderExecutedData struct {
	OrderId     int64   `json:"orderId"`     // 訂單 ID
	Symbol      string  `json:"symbol"`      // 交易對
	Side        string  `json:"side"`        // 買入或賣出
	Quantity    float64 `json:"quantity"`    // 數量
	Price       float64 `json:"price"`       // 成交價格
	TotalAmount float64 `json:"totalAmount"` // 總金額
	Status      string  `json:"status"`      // 訂單狀態
}

// LimitOrderFilledData 限價單成交數據
type LimitOrderFilledData struct {
	OrderId       int64   `json:"orderId"`       // 訂單 ID
	Symbol        string  `json:"symbol"`        // 交易對
	Side          string  `json:"side"`          // 買入或賣出
	LimitPrice    float64 `json:"limitPrice"`    // 限價
	Quantity      float64 `json:"quantity"`      // 數量
	ExecutedPrice float64 `json:"executedPrice"` // 執行價格
	TotalAmount   float64 `json:"totalAmount"`   // 總金額
	Status        string  `json:"status"`        // 訂單狀態
}

// LeveragePositionOpenedData 槓桿位置開倉數據
type LeveragePositionOpenedData struct {
	PositionId       int64   `json:"positionId"`       // 位置 ID
	Symbol           string  `json:"symbol"`           // 交易對
	Side             string  `json:"side"`             // LONG 或 SHORT
	Leverage         int     `json:"leverage"`         // 槓桿倍數
	Quantity         float64 `json:"quantity"`         // 數量
	EntryPrice       float64 `json:"entryPrice"`       // 開倉價格
	Margin           float64 `json:"margin"`           // 保證金
	LiquidationPrice float64 `json:"liquidationPrice"` // 爆倉價格
	Status           string  `json:"status"`           // 位置狀態
}

// LeveragePositionClosedData 槓桿位置平倉數據
type LeveragePositionClosedData struct {
	PositionId    int64   `json:"positionId"`    // 位置 ID
	Symbol        string  `json:"symbol"`        // 交易對
	Side          string  `json:"side"`          // LONG 或 SHORT
	Leverage      int     `json:"leverage"`      // 槓桿倍數
	EntryPrice    float64 `json:"entryPrice"`    // 開倉價格
	ExitPrice     float64 `json:"exitPrice"`     // 平倉價格
	Quantity      float64 `json:"quantity"`      // 數量
	PnL           float64 `json:"pnl"`           // 損益
	PnLPercentage float64 `json:"pnlPercentage"` // 損益百分比
	Status        string  `json:"status"`        // 位置狀態
}

// NewOrderExecutedMessage 創建訂單成交消息
func NewOrderExecutedMessage(order *Order) *WSMessage {
	return &WSMessage{
		Type:      WSMessageTypeOrderExecuted,
		Timestamp: time.Now(),
		Data: &OrderExecutedData{
			OrderId:     order.Id,
			Symbol:      order.Symbol,
			Side:        string(order.Side),
			Quantity:    order.Quantity,
			Price:       order.Price,
			TotalAmount: order.TotalAmount,
			Status:      string(order.Status),
		},
	}
}

// NewLimitOrderFilledMessage 創建限價單成交消息
func NewLimitOrderFilledMessage(orderId int64, symbol string, side OrderSide, limitPrice, executedPrice, quantity, totalAmount float64) *WSMessage {
	return &WSMessage{
		Type:      WSMessageTypeLimitOrderFilled,
		Timestamp: time.Now(),
		Data: &LimitOrderFilledData{
			OrderId:       orderId,
			Symbol:        symbol,
			Side:          string(side),
			LimitPrice:    limitPrice,
			ExecutedPrice: executedPrice,
			Quantity:      quantity,
			TotalAmount:   totalAmount,
			Status:        string(OrderStatusCompleted),
		},
	}
}

// NewLeveragePositionOpenedMessage 創建槓桿位置開倉消息
func NewLeveragePositionOpenedMessage(position *LeveragePosition) *WSMessage {
	return &WSMessage{
		Type:      WSMessageTypeLeveragePositionOpened,
		Timestamp: time.Now(),
		Data: &LeveragePositionOpenedData{
			PositionId:       position.Id,
			Symbol:           position.Symbol,
			Side:             string(position.Side),
			Leverage:         position.Leverage,
			Quantity:         position.Quantity,
			EntryPrice:       position.EntryPrice,
			Margin:           position.Margin,
			LiquidationPrice: position.LiquidationPrice,
			Status:           string(position.Status),
		},
	}
}

// NewLeveragePositionClosedMessage 創建槓桿位置平倉消息
func NewLeveragePositionClosedMessage(position *LeveragePosition, exitPrice float64) *WSMessage {
	pnl := position.CalculateUnrealizedPnL(exitPrice)
	pnlPercentage := (pnl / position.Margin) * 100

	return &WSMessage{
		Type:      WSMessageTypeLeveragePositionClosed,
		Timestamp: time.Now(),
		Data: &LeveragePositionClosedData{
			PositionId:    position.Id,
			Symbol:        position.Symbol,
			Side:          string(position.Side),
			Leverage:      position.Leverage,
			EntryPrice:    position.EntryPrice,
			ExitPrice:     exitPrice,
			Quantity:      position.Quantity,
			PnL:           pnl,
			PnLPercentage: pnlPercentage,
			Status:        string(position.Status),
		},
	}
}

// ToJSON 將消息轉換為 JSON
func (m *WSMessage) ToJSON() []byte {
	data, _ := json.Marshal(m)
	return data
}
