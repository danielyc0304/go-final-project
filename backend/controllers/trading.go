package controllers

import (
	"backend/models"
	"backend/services"
	"backend/utils"
	"encoding/json"
	"strconv"

	"github.com/beego/beego/v2/server/web"
)

type TradingController struct {
	web.Controller
}

// PlaceOrderRequest 下單請求
type PlaceOrderRequest struct {
	Symbol     string   `json:"symbol" valid:"Required"`   // 交易對：BTCUSDT, ETHUSDT, SOLUSDT
	Type       string   `json:"type" valid:"Required"`     // MARKET 或 LIMIT
	Side       string   `json:"side" valid:"Required"`     // BUY 或 SELL
	Quantity   float64  `json:"quantity" valid:"Required"` // 數量
	LimitPrice *float64 `json:"limitPrice,omitempty"`      // 限價（僅限價單需要）
}

// PlaceOrder 下單（支援市價單和限價單）
// @Title PlaceOrder
// @Description 執行市價單或限價單買入/賣出
// @Param	Authorization	header	string	true	"Bearer {token}"
// @Param	body			body	PlaceOrderRequest	true	"訂單資訊"
// @Success 200 {object} models.Order
// @Failure 400 Bad request
// @Failure 401 Unauthorized
// @Failure 500 Internal server error
// @router /order [post]
func (c *TradingController) PlaceOrder() {
	// 1. 驗證 JWT
	userId, err := utils.ValidateJWT(c.Ctx.Request)
	if err != nil {
		utils.RespondError(c.Ctx, 401, "Unauthorized: "+err.Error())
		return
	}

	// 2. 解析請求
	var req PlaceOrderRequest
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		utils.RespondError(c.Ctx, 400, "Invalid request body")
		return
	}

	// 3. 驗證輸入
	if req.Quantity <= 0 {
		utils.RespondError(c.Ctx, 400, "Quantity must be positive")
		return
	}

	var side models.OrderSide
	if req.Side == "BUY" {
		side = models.OrderSideBuy
	} else if req.Side == "SELL" {
		side = models.OrderSideSell
	} else {
		utils.RespondError(c.Ctx, 400, "Invalid side, must be BUY or SELL")
		return
	}

	var order *models.Order

	// 4. 根據訂單類型執行
	if req.Type == "MARKET" {
		// 市價單
		order, err = services.PlaceMarketOrder(userId, req.Symbol, side, req.Quantity)
	} else if req.Type == "LIMIT" {
		// 限價單
		if req.LimitPrice == nil || *req.LimitPrice <= 0 {
			utils.RespondError(c.Ctx, 400, "Limit price is required and must be positive for limit orders")
			return
		}
		order, err = services.PlaceLimitOrder(userId, req.Symbol, side, req.Quantity, *req.LimitPrice)
	} else {
		utils.RespondError(c.Ctx, 400, "Invalid order type, must be MARKET or LIMIT")
		return
	}

	if err != nil {
		utils.RespondError(c.Ctx, 400, "Failed to place order: "+err.Error())
		return
	}

	// 5. 返回結果
	utils.RespondJSON(c.Ctx, 200, map[string]interface{}{
		"success": true,
		"message": "Order placed successfully",
		"order":   order,
	})
}

// GetOrders 查詢使用者訂單
// @Title GetOrders
// @Description 查詢使用者的訂單歷史
// @Param	Authorization	header	string	true	"Bearer {token}"
// @Param	symbol			query	string	false	"交易對篩選"
// @Param	limit			query	int		false	"每頁數量（預設20）"
// @Param	offset			query	int		false	"偏移量（預設0）"
// @Success 200 {array} models.Order
// @Failure 401 Unauthorized
// @Failure 500 Internal server error
// @router /orders [get]
func (c *TradingController) GetOrders() {
	// 1. 驗證 JWT
	userId, err := utils.ValidateJWT(c.Ctx.Request)
	if err != nil {
		utils.RespondError(c.Ctx, 401, "Unauthorized: "+err.Error())
		return
	}

	// 2. 解析查詢參數
	symbol := c.GetString("symbol", "")
	limit, _ := strconv.Atoi(c.GetString("limit", "20"))
	offset, _ := strconv.Atoi(c.GetString("offset", "0"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// 3. 查詢訂單
	var orders []*models.Order
	if symbol != "" {
		orders, err = models.GetOrdersByUserAndSymbol(userId, symbol, limit, offset)
	} else {
		orders, err = models.GetOrdersByUser(userId, limit, offset)
	}

	if err != nil {
		utils.RespondError(c.Ctx, 500, "Failed to get orders: "+err.Error())
		return
	}

	// 4. 返回結果
	utils.RespondJSON(c.Ctx, 200, map[string]interface{}{
		"success": true,
		"orders":  orders,
		"count":   len(orders),
	})
}

// GetWallets 查詢使用者錢包
// @Title GetWallets
// @Description 查詢使用者的所有錢包餘額
// @Param	Authorization	header	string	true	"Bearer {token}"
// @Success 200 {array} models.Wallet
// @Failure 401 Unauthorized
// @Failure 500 Internal server error
// @router /wallets [get]
func (c *TradingController) GetWallets() {
	// 1. 驗證 JWT
	userId, err := utils.ValidateJWT(c.Ctx.Request)
	if err != nil {
		utils.RespondError(c.Ctx, 401, "Unauthorized: "+err.Error())
		return
	}

	// 2. 查詢錢包
	wallets, err := models.GetAllWalletsByUser(userId)
	if err != nil {
		utils.RespondError(c.Ctx, 500, "Failed to get wallets: "+err.Error())
		return
	}

	// 3. 如果沒有錢包，初始化預設錢包
	if len(wallets) == 0 {
		err = models.InitializeDefaultWallets(userId)
		if err != nil {
			utils.RespondError(c.Ctx, 500, "Failed to initialize wallets: "+err.Error())
			return
		}
		wallets, _ = models.GetAllWalletsByUser(userId)
	}

	// 4. 返回結果
	utils.RespondJSON(c.Ctx, 200, map[string]interface{}{
		"success": true,
		"wallets": wallets,
	})
}

// GetTransactions 查詢交易記錄
// @Title GetTransactions
// @Description 查詢使用者的交易記錄
// @Param	Authorization	header	string	true	"Bearer {token}"
// @Param	symbol			query	string	false	"幣種篩選"
// @Param	limit			query	int		false	"每頁數量（預設20）"
// @Param	offset			query	int		false	"偏移量（預設0）"
// @Success 200 {array} models.Transaction
// @Failure 401 Unauthorized
// @Failure 500 Internal server error
// @router /transactions [get]
func (c *TradingController) GetTransactions() {
	// 1. 驗證 JWT
	userId, err := utils.ValidateJWT(c.Ctx.Request)
	if err != nil {
		utils.RespondError(c.Ctx, 401, "Unauthorized: "+err.Error())
		return
	}

	// 2. 解析查詢參數
	symbol := c.GetString("symbol", "")
	limit, _ := strconv.Atoi(c.GetString("limit", "20"))
	offset, _ := strconv.Atoi(c.GetString("offset", "0"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// 3. 查詢交易記錄
	var transactions []*models.Transaction
	if symbol != "" {
		transactions, err = models.GetTransactionsByUserAndSymbol(userId, symbol, limit, offset)
	} else {
		transactions, err = models.GetTransactionsByUser(userId, limit, offset)
	}

	if err != nil {
		utils.RespondError(c.Ctx, 500, "Failed to get transactions: "+err.Error())
		return
	}

	// 4. 返回結果
	utils.RespondJSON(c.Ctx, 200, map[string]interface{}{
		"success":      true,
		"transactions": transactions,
		"count":        len(transactions),
	})
}

// GetPrices 取得當前市價
// @Title GetPrices
// @Description 取得所有支援交易對的當前價格
// @Success 200 {object} map[string]float64
// @router /prices [get]
func (c *TradingController) GetPrices() {
	prices := services.GlobalPriceCache.GetAllPrices()

	utils.RespondJSON(c.Ctx, 200, map[string]interface{}{
		"success": true,
		"prices":  prices,
	})
}

// CancelOrder 取消訂單
// @Title CancelOrder
// @Description 取消待處理的訂單（僅限價單可取消）
// @Param	Authorization	header	string	true	"Bearer {token}"
// @Param	id				path	int		true	"訂單 ID"
// @Success 200 {string} string "Order canceled successfully"
// @Failure 400 Bad request
// @Failure 401 Unauthorized
// @Failure 404 Order not found
// @router /order/:id/cancel [post]
func (c *TradingController) CancelOrder() {
	// 1. 驗證 JWT
	userId, err := utils.ValidateJWT(c.Ctx.Request)
	if err != nil {
		utils.RespondError(c.Ctx, 401, "Unauthorized: "+err.Error())
		return
	}

	// 2. 解析訂單 ID
	orderIdStr := c.Ctx.Input.Param(":id")
	orderId, err := strconv.ParseInt(orderIdStr, 10, 64)
	if err != nil {
		utils.RespondError(c.Ctx, 400, "Invalid order ID")
		return
	}

	// 3. 取消訂單
	err = models.CancelOrder(orderId, userId)
	if err != nil {
		if err.Error() == "unauthorized: order does not belong to user" {
			utils.RespondError(c.Ctx, 403, err.Error())
		} else if err.Error() == "order cannot be canceled" {
			utils.RespondError(c.Ctx, 400, err.Error())
		} else {
			utils.RespondError(c.Ctx, 404, "Order not found")
		}
		return
	}

	// 4. 從撮合器中移除
	services.GlobalLimitOrderMatcher.RemoveOrder(orderId)

	// 5. 返回結果
	utils.RespondJSON(c.Ctx, 200, map[string]interface{}{
		"success": true,
		"message": "Order canceled successfully",
	})
}
