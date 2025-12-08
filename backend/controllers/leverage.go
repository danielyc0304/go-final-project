package controllers

import (
	"backend/models"
	"backend/services"
	"backend/utils"
	"encoding/json"
	"strconv"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type LeverageController struct {
	web.Controller
}

// OpenPositionRequest 開倉請求
type OpenPositionRequest struct {
	Symbol     string              `json:"symbol" valid:"Required"`    // 交易對：BTCUSDT, ETHUSDT, SOLUSDT
	Side       models.PositionSide `json:"side" valid:"Required"`      // LONG 或 SHORT
	Leverage   int                 `json:"leverage" valid:"Required"`  // 槓桿倍數 1-10
	Quantity   float64             `json:"quantity" valid:"Required"`  // 數量
	OrderType  models.OrderType    `json:"orderType" valid:"Required"` // MARKET 或 LIMIT
	LimitPrice *float64            `json:"limitPrice,omitempty"`       // 限價（僅限價單需要）
}

// OpenPosition 開槓桿倉位
// @Title OpenPosition
// @Description 開設槓桿倉位（做多/做空）
// @Param	Authorization	header	string	true	"Bearer {token}"
// @Param	body			body	OpenPositionRequest	true	"開倉資訊"
// @Success 200 {object} models.LeveragePosition
// @Failure 400 Bad request
// @Failure 401 Unauthorized
// @Failure 500 Internal server error
// @router /position/open [post]
func (c *LeverageController) OpenPosition() {
	// 1. 驗證 JWT
	userId, err := utils.ValidateJWT(c.Ctx.Request)
	if err != nil {
		utils.RespondError(c.Ctx, 401, "Unauthorized: "+err.Error())
		return
	}

	// 2. 解析請求
	var req OpenPositionRequest
	if err = json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		utils.RespondError(c.Ctx, 400, "Invalid request body")
		return
	}

	// 3. 驗證輸入
	if req.Quantity <= 0 {
		utils.RespondError(c.Ctx, 400, "Quantity must be positive")
		return
	}

	if req.Leverage < 1 || req.Leverage > 10 {
		utils.RespondError(c.Ctx, 400, "Leverage must be between 1 and 10")
		return
	}

	if req.Side != models.PositionSideLong && req.Side != models.PositionSideShort {
		utils.RespondError(c.Ctx, 400, "Side must be LONG or SHORT")
		return
	}

	// 驗證訂單類型
	if req.OrderType != models.OrderTypeMarket && req.OrderType != models.OrderTypeLimit {
		utils.RespondError(c.Ctx, 400, "OrderType must be MARKET or LIMIT")
		return
	}

	// 限價單需要限價
	if req.OrderType == models.OrderTypeLimit && req.LimitPrice == nil {
		utils.RespondError(c.Ctx, 400, "LimitPrice is required for limit orders")
		return
	}

	if req.OrderType == models.OrderTypeLimit && *req.LimitPrice <= 0 {
		utils.RespondError(c.Ctx, 400, "LimitPrice must be positive")
		return
	}

	// 4. 開倉
	var position *models.LeveragePosition

	if req.OrderType == models.OrderTypeMarket {
		position, err = services.OpenLeveragePositionMarket(userId, req.Symbol, req.Side, req.Leverage, req.Quantity)
	} else {
		position, err = services.OpenLeveragePositionLimit(userId, req.Symbol, req.Side, req.Leverage, req.Quantity, *req.LimitPrice)
	}
	if err != nil {
		utils.RespondError(c.Ctx, 400, "Failed to open position: "+err.Error())
		return
	}

	// 5. 返回結果
	utils.RespondJSON(c.Ctx, 200, map[string]interface{}{
		"success":  true,
		"message":  "Position opened successfully",
		"position": position,
	})
}

// ClosePosition 平槓桿倉位
// @Title ClosePosition
// @Description 平倉（關閉槓桿倉位）
// @Param	Authorization	header	string	true	"Bearer {token}"
// @Param	id				path	int		true	"倉位 ID"
// @Success 200 {object} models.LeveragePosition
// @Failure 400 Bad request
// @Failure 401 Unauthorized
// @Failure 404 Position not found
// @router /position/:id/close [post]
func (c *LeverageController) ClosePosition() {
	// 1. 驗證 JWT
	userId, err := utils.ValidateJWT(c.Ctx.Request)
	if err != nil {
		utils.RespondError(c.Ctx, 401, "Unauthorized: "+err.Error())
		return
	}

	// 2. 解析倉位 ID
	positionIdStr := c.Ctx.Input.Param(":id")
	positionId, err := strconv.ParseInt(positionIdStr, 10, 64)
	if err != nil {
		utils.RespondError(c.Ctx, 400, "Invalid position ID")
		return
	}

	// 3. 平倉
	position, err := services.CloseLeveragePosition(userId, positionId)
	if err != nil {
		if err.Error() == "unauthorized: position does not belong to user" {
			utils.RespondError(c.Ctx, 403, err.Error())
		} else if err.Error() == "position is not open" {
			utils.RespondError(c.Ctx, 400, err.Error())
		} else if err.Error() == "position not found" {
			utils.RespondError(c.Ctx, 404, err.Error())
		} else {
			utils.RespondError(c.Ctx, 500, "Failed to close position: "+err.Error())
		}
		return
	}

	// 4. 返回結果
	utils.RespondJSON(c.Ctx, 200, map[string]interface{}{
		"success":  true,
		"message":  "Position closed successfully",
		"position": position,
	})
}

// GetOpenPositions 查詢持倉
// @Title GetOpenPositions
// @Description 查詢使用者的所有持倉
// @Param	Authorization	header	string	true	"Bearer {token}"
// @Success 200 {array} models.LeveragePosition
// @Failure 401 Unauthorized
// @Failure 500 Internal server error
// @router /positions/open [get]
func (c *LeverageController) GetOpenPositions() {
	// 1. 驗證 JWT
	userId, err := utils.ValidateJWT(c.Ctx.Request)
	if err != nil {
		utils.RespondError(c.Ctx, 401, "Unauthorized: "+err.Error())
		return
	}

	// 2. 查詢持倉
	positions, err := models.GetOpenPositionsByUser(userId)
	if err != nil {
		utils.RespondError(c.Ctx, 500, "Failed to get positions: "+err.Error())
		return
	}

	// 3. 更新未實現盈虧
	for _, position := range positions {
		currentPrice, ok := services.GlobalPriceCache.GetPrice(position.Symbol)
		if ok {
			position.UnrealizedPnL = position.CalculateUnrealizedPnL(currentPrice)
		}
	}

	// 4. 返回結果
	utils.RespondJSON(c.Ctx, 200, map[string]interface{}{
		"success":   true,
		"positions": positions,
		"count":     len(positions),
	})
}

// GetPositionHistory 查詢倉位歷史
// @Title GetPositionHistory
// @Description 查詢使用者的倉位歷史（包含已平倉）
// @Param	Authorization	header	string	true	"Bearer {token}"
// @Param	symbol			query	string	false	"交易對篩選"
// @Param	limit			query	int		false	"每頁數量（預設20）"
// @Param	offset			query	int		false	"偏移量（預設0）"
// @Success 200 {array} models.LeveragePosition
// @Failure 401 Unauthorized
// @Failure 500 Internal server error
// @router /positions/history [get]
func (c *LeverageController) GetPositionHistory() {
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

	// 3. 查詢倉位歷史
	var positions []*models.LeveragePosition
	if symbol != "" {
		positions, err = models.GetPositionsByUserAndSymbol(userId, symbol, limit, offset)
	} else {
		positions, err = models.GetAllPositionsByUser(userId, limit, offset)
	}

	if err != nil {
		utils.RespondError(c.Ctx, 500, "Failed to get positions: "+err.Error())
		return
	}

	// 4. 返回結果
	utils.RespondJSON(c.Ctx, 200, map[string]interface{}{
		"success":   true,
		"positions": positions,
		"count":     len(positions),
	})
}

// GetPositionDetail 查詢倉位詳情
// @Title GetPositionDetail
// @Description 查詢單個倉位的詳細資訊
// @Param	Authorization	header	string	true	"Bearer {token}"
// @Param	id				path	int		true	"倉位 ID"
// @Success 200 {object} models.LeveragePosition
// @Failure 401 Unauthorized
// @Failure 404 Position not found
// @router /position/:id [get]
func (c *LeverageController) GetPositionDetail() {
	// 1. 驗證 JWT
	userId, err := utils.ValidateJWT(c.Ctx.Request)
	if err != nil {
		utils.RespondError(c.Ctx, 401, "Unauthorized: "+err.Error())
		return
	}

	// 2. 解析倉位 ID
	positionIdStr := c.Ctx.Input.Param(":id")
	positionId, err := strconv.ParseInt(positionIdStr, 10, 64)
	if err != nil {
		utils.RespondError(c.Ctx, 400, "Invalid position ID")
		return
	}

	// 3. 查詢倉位
	position, err := models.GetPositionById(positionId)
	if err != nil {
		utils.RespondError(c.Ctx, 404, "Position not found")
		return
	}

	// 4. 驗證所有權
	o := orm.NewOrm()
	o.LoadRelated(position, "User")

	if position.User.Id != userId {
		utils.RespondError(c.Ctx, 403, "Unauthorized: position does not belong to user")
		return
	}

	// 5. 更新未實現盈虧（如果倉位還開著）
	if position.Status == models.PositionStatusOpen {
		currentPrice, ok := services.GlobalPriceCache.GetPrice(position.Symbol)
		if ok {
			position.UnrealizedPnL = position.CalculateUnrealizedPnL(currentPrice)
		}
	}

	// 6. 返回結果
	utils.RespondJSON(c.Ctx, 200, map[string]interface{}{
		"success":  true,
		"position": position,
	})
}
