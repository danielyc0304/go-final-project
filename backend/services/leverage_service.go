package services

import (
	"backend/hub"
	"backend/models"
	"errors"
	"fmt"
	"log"

	"github.com/beego/beego/v2/client/orm"
)

// OpenLeveragePositionMarket 用市價單開槓桿倉位
func OpenLeveragePositionMarket(userId int64, symbol string, side models.PositionSide, leverage int, quantity float64) (*models.LeveragePosition, error) {
	return OpenLeveragePosition(userId, symbol, side, leverage, quantity)
}

// OpenLeveragePositionLimit 用限價單開槓桿倉位
func OpenLeveragePositionLimit(userId int64, symbol string, side models.PositionSide, leverage int, quantity float64, limitPrice float64) (*models.LeveragePosition, error) {
	// 1. 驗證輸入
	if quantity <= 0 {
		return nil, errors.New("quantity must be positive")
	}

	if leverage < 1 || leverage > 100 {
		return nil, errors.New("leverage must be between 1 and 100")
	}

	if limitPrice <= 0 {
		return nil, errors.New("limit price must be positive")
	}

	_, quote, err := models.ParseSymbol(symbol)
	if err != nil {
		return nil, err
	}

	if quote != "USDT" {
		return nil, errors.New("only USDT pairs are supported for leverage trading")
	}

	// 2. 建立限價訂單
	order, err := models.CreateOrder(userId, symbol, models.OrderTypeLimit,
		func() models.OrderSide {
			if side == models.PositionSideLong {
				return models.OrderSideBuy
			} else {
				return models.OrderSideSell
			}
		}(), quantity, &limitPrice)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %v", err)
	}

	// 3. 計算所需保證金
	margin := models.CalculateRequiredMargin(limitPrice, quantity, leverage)

	// 4. 開始資料庫交易
	o := orm.NewOrm()
	to, err := o.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %v", err)
	}

	shouldRollback := true
	defer func() {
		if shouldRollback {
			to.Rollback()
		}
	}()

	// 5. 檢查並扣除保證金（從 USDT 錢包）
	wallet, err := models.GetWalletByUserAndSymbol(userId, "USDT")
	if err != nil {
		return nil, errors.New("USDT wallet not found")
	}

	if wallet.Balance < margin {
		return nil, fmt.Errorf("insufficient USDT balance: required %.2f, available %.2f", margin, wallet.Balance)
	}

	// 扣除保證金
	err = models.UpdateBalance(to, wallet.Id, -margin, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to deduct margin: %v", err)
	}

	// 6. 創建槓桿倉位（狀態為 PENDING，等待限價單成交）
	position := &models.LeveragePosition{
		User:       &models.User{Id: userId},
		Order:      order,
		Symbol:     symbol,
		Side:       side,
		Leverage:   leverage,
		EntryPrice: limitPrice, // 暫時設為限價
		Quantity:   quantity,
		Margin:     margin,
		Status:     models.PositionStatusOpen, // 先設為 OPEN，會在限價單成交時更新
	}

	// 計算爆倉價格
	position.LiquidationPrice = position.CalculateLiquidationPrice()

	_, err = to.Insert(position)
	if err != nil {
		return nil, fmt.Errorf("failed to create position: %v", err)
	}

	// 7. 加入限價單撮合器監控（檢查價格，可能立即執行）
	currentPrice, ok := GlobalPriceCache.GetPrice(symbol)
	shouldExecuteImmediately := false

	if ok {
		if side == models.PositionSideLong {
			// 買入限價單：當市價 <= 限價時可以立即成交
			if currentPrice <= limitPrice {
				shouldExecuteImmediately = true
			}
		} else {
			// 賣出限價單：當市價 >= 限價時可以立即成交
			if currentPrice >= limitPrice {
				shouldExecuteImmediately = true
			}
		}
	}

	if shouldExecuteImmediately {
		log.Printf("Leverage limit order #%d will be executed immediately: %s %s at %.2f (current price: %.2f)",
			order.Id, side, symbol, limitPrice, currentPrice)

		// 執行訂單（使用當前市價，但不超過限價）
		executionPrice := currentPrice
		if side == models.PositionSideLong && executionPrice > limitPrice {
			executionPrice = limitPrice
		} else if side == models.PositionSideShort && executionPrice < limitPrice {
			executionPrice = limitPrice
		}

		err := GlobalLimitOrderMatcher.ExecuteLimitOrder(order, executionPrice)
		if err != nil {
			log.Printf("Failed to execute leverage limit order #%d immediately: %v", order.Id, err)
			// 如果立即執行失敗，改為加入待處理列表，稍後重試
			GlobalLimitOrderMatcher.AddOrder(order)
		}
	} else {
		log.Printf("Leverage limit order #%d added to matcher: %s %s at %.2f (current price: %.2f)",
			order.Id, side, symbol, limitPrice, currentPrice)
		GlobalLimitOrderMatcher.AddOrder(order)
	}

	// 8. 記錄交易
	transactionType := models.TransactionTypeMarginDeposit
	description := fmt.Sprintf("Open %s position #%d with %dx leverage (Limit Order)", side, position.Id, leverage)
	_, err = models.CreateTransaction(to, userId, nil, transactionType, "USDT", -margin,
		wallet.Balance+margin, wallet.Balance, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %v", err)
	}

	// 9. 提交交易
	err = to.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	shouldRollback = false

	log.Printf("Leverage position (limit order) created: User=%d, Symbol=%s, Side=%s, Leverage=%dx, Quantity=%.8f, LimitPrice=%.2f, Margin=%.2f",
		userId, symbol, side, leverage, quantity, limitPrice, margin)

	// 發送 WebSocket 通知給用戶
	message := models.NewLeveragePositionOpenedMessage(position)
	hub.GlobalHub.BroadcastToUser(userId, message.ToJSON())

	return position, nil
}

// OpenLeveragePosition 開槓桿倉位
func OpenLeveragePosition(userId int64, symbol string, side models.PositionSide, leverage int, quantity float64) (*models.LeveragePosition, error) {
	// 1. 驗證輸入
	if quantity <= 0 {
		return nil, errors.New("quantity must be positive")
	}

	if leverage < 1 || leverage > 10 {
		return nil, errors.New("leverage must be between 1 and 10")
	}

	_, quote, err := models.ParseSymbol(symbol)
	if err != nil {
		return nil, err
	}

	if quote != "USDT" {
		return nil, errors.New("only USDT pairs are supported for leverage trading")
	}

	// 2. 獲取當前市價
	currentPrice, ok := GlobalPriceCache.GetPrice(symbol)
	if !ok {
		return nil, fmt.Errorf("price not available for %s", symbol)
	}

	// 3. 計算所需保證金
	margin := models.CalculateRequiredMargin(currentPrice, quantity, leverage)

	// 4. 開始資料庫交易
	o := orm.NewOrm()
	to, err := o.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %v", err)
	}

	shouldRollback := true
	defer func() {
		if shouldRollback {
			to.Rollback()
		}
	}()

	// 5. 檢查並扣除保證金（從 USDT 錢包）
	wallet, err := models.GetWalletByUserAndSymbol(userId, "USDT")
	if err != nil {
		return nil, errors.New("USDT wallet not found")
	}

	if wallet.Balance < margin {
		return nil, fmt.Errorf("insufficient USDT balance: required %.2f, available %.2f", margin, wallet.Balance)
	}

	// 扣除保證金
	err = models.UpdateBalance(to, wallet.Id, -margin, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to deduct margin: %v", err)
	}

	// 6. 創建槓桿倉位
	position, err := models.CreateLeveragePosition(userId, symbol, side, leverage, currentPrice, quantity, margin)
	if err != nil {
		return nil, fmt.Errorf("failed to create position: %v", err)
	}

	// 7. 記錄交易
	transactionType := models.TransactionTypeMarginDeposit
	description := fmt.Sprintf("Open %s position #%d with %dx leverage", side, position.Id, leverage)
	_, err = models.CreateTransaction(to, userId, nil, transactionType, "USDT", -margin,
		wallet.Balance+margin, wallet.Balance, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %v", err)
	}

	// 8. 提交交易
	err = to.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	shouldRollback = false

	log.Printf("Leverage position opened: User=%d, Symbol=%s, Side=%s, Leverage=%dx, Quantity=%.8f, EntryPrice=%.2f, Margin=%.2f",
		userId, symbol, side, leverage, quantity, currentPrice, margin)

	// 發送 WebSocket 通知給用戶
	message := models.NewLeveragePositionOpenedMessage(position)
	hub.GlobalHub.BroadcastToUser(userId, message.ToJSON())

	return position, nil
}

// CloseLeveragePosition 平槓桿倉位
func CloseLeveragePosition(userId int64, positionId int64) (*models.LeveragePosition, error) {
	// 1. 獲取倉位
	position, err := models.GetPositionById(positionId)
	if err != nil {
		return nil, err
	}

	// 2. 獲取當前市價
	currentPrice, ok := GlobalPriceCache.GetPrice(position.Symbol)
	if !ok {
		return nil, fmt.Errorf("price not available for %s", position.Symbol)
	}

	// 3. 開始資料庫交易
	o := orm.NewOrm()
	to, err := o.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %v", err)
	}

	shouldRollback := true
	defer func() {
		if shouldRollback {
			to.Rollback()
		}
	}()

	// 4. 計算盈虧
	pnl := position.CalculateUnrealizedPnL(currentPrice)

	// 5. 平倉
	err = models.ClosePosition(positionId, userId, currentPrice)
	if err != nil {
		return nil, err
	}

	// 6. 返還保證金 + 盈虧到 USDT 錢包
	returnAmount := position.Margin + pnl

	wallet, err := models.GetWalletByUserAndSymbol(userId, "USDT")
	if err != nil {
		return nil, errors.New("USDT wallet not found")
	}

	if returnAmount > 0 {
		err = models.UpdateBalance(to, wallet.Id, returnAmount, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to return funds: %v", err)
		}
	}

	// 7. 記錄交易
	transactionType := models.TransactionTypeMarginWithdraw
	description := fmt.Sprintf("Close %s position #%d: PnL %.2f USDT", position.Side, position.Id, pnl)
	_, err = models.CreateTransaction(to, userId, nil, transactionType, "USDT", returnAmount,
		wallet.Balance, wallet.Balance+returnAmount, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %v", err)
	}

	// 8. 提交交易
	err = to.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	shouldRollback = false

	// 9. 重新讀取倉位以獲取更新後的數據
	position, _ = models.GetPositionById(positionId)

	log.Printf("Leverage position closed: User=%d, Position=#%d, ExitPrice=%.2f, PnL=%.2f",
		userId, positionId, currentPrice, pnl)

	// 發送 WebSocket 通知給用戶
	message := models.NewLeveragePositionClosedMessage(position, currentPrice)
	hub.GlobalHub.BroadcastToUser(userId, message.ToJSON())

	return position, nil
}

// CheckAndLiquidatePositions 檢查並執行爆倉
func CheckAndLiquidatePositions() {
	positions, err := models.GetAllOpenPositions()
	if err != nil {
		log.Printf("Failed to get open positions: %v", err)
		return
	}

	for _, position := range positions {
		currentPrice, ok := GlobalPriceCache.GetPrice(position.Symbol)
		if !ok {
			continue
		}

		// 檢查是否觸發爆倉
		if position.IsLiquidated(currentPrice) {
			log.Printf("Liquidating position #%d: User=%d, Symbol=%s, Side=%s, LiqPrice=%.2f, CurrentPrice=%.2f",
				position.Id, position.User.Id, position.Symbol, position.Side, position.LiquidationPrice, currentPrice)

			err := liquidatePosition(position)
			if err != nil {
				log.Printf("Failed to liquidate position #%d: %v", position.Id, err)
			}
		}
	}
}

// liquidatePosition 執行爆倉
func liquidatePosition(position *models.LeveragePosition) error {
	o := orm.NewOrm()
	to, err := o.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}

	shouldRollback := true
	defer func() {
		if shouldRollback {
			to.Rollback()
		}
	}()

	// 載入 User
	o.LoadRelated(position, "User")
	userId := position.User.Id

	// 平倉（爆倉）
	err = models.LiquidatePosition(position.Id)
	if err != nil {
		return err
	}

	// 記錄交易（保證金全部虧損）
	transactionType := models.TransactionTypeLiquidation
	description := fmt.Sprintf("Position #%d liquidated at %.2f", position.Id, position.LiquidationPrice)
	_, err = models.CreateTransaction(to, userId, nil, transactionType, "USDT", 0, 0, 0, description)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %v", err)
	}

	err = to.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	shouldRollback = false

	log.Printf("Position #%d liquidated successfully", position.Id)

	// 發送 WebSocket 通知給用戶
	message := models.NewLeveragePositionClosedMessage(position, position.LiquidationPrice)
	hub.GlobalHub.BroadcastToUser(userId, message.ToJSON())

	return nil
}

// UpdateAllPositionsPnL 更新所有持倉的盈虧
func UpdateAllPositionsPnL() {
	// 這個函數可以定期調用來更新所有持倉的未實現盈虧
	positions, err := models.GetAllOpenPositions()
	if err != nil {
		return
	}

	for _, position := range positions {
		currentPrice, ok := GlobalPriceCache.GetPrice(position.Symbol)
		if !ok {
			continue
		}

		models.UpdatePositionPnL(position, currentPrice)
	}
}
