package services

import (
	"backend/hub"
	"backend/models"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// LimitOrderMatcher 限價單撮合器
type LimitOrderMatcher struct {
	mu            sync.RWMutex
	isRunning     bool
	stopChan      chan struct{}
	checkInterval time.Duration
	pendingOrders map[int64]*models.Order // orderId -> Order
}

var GlobalLimitOrderMatcher *LimitOrderMatcher

func init() {
	GlobalLimitOrderMatcher = &LimitOrderMatcher{
		checkInterval: 1 * time.Second, // 每秒檢查一次
		pendingOrders: make(map[int64]*models.Order),
		stopChan:      make(chan struct{}),
	}
}

// Start 啟動限價單撮合服務
func (m *LimitOrderMatcher) Start() {
	m.mu.Lock()
	if m.isRunning {
		m.mu.Unlock()
		return
	}
	m.isRunning = true
	m.mu.Unlock()

	log.Println("Limit order matcher started")

	// 載入現有的待處理限價單
	m.loadPendingOrders()

	// 啟動監控循環
	go m.run()
}

// Stop 停止限價單撮合服務
func (m *LimitOrderMatcher) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return
	}

	m.isRunning = false
	close(m.stopChan)
	log.Println("Limit order matcher stopped")
}

// loadPendingOrders 從資料庫載入待處理的限價單
func (m *LimitOrderMatcher) loadPendingOrders() {
	orders, err := models.GetPendingLimitOrders()
	if err != nil {
		log.Printf("Failed to load pending limit orders: %v", err)
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, order := range orders {
		m.pendingOrders[order.Id] = order
	}

	log.Printf("Loaded %d pending limit orders", len(orders))
}

// run 主要監控循環
func (m *LimitOrderMatcher) run() {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.checkAndExecuteOrders()
		}
	}
}

// checkAndExecuteOrders 檢查並執行符合條件的限價單
func (m *LimitOrderMatcher) checkAndExecuteOrders() {
	m.mu.RLock()
	ordersCopy := make([]*models.Order, 0, len(m.pendingOrders))
	for _, order := range m.pendingOrders {
		ordersCopy = append(ordersCopy, order)
	}
	m.mu.RUnlock()

	for _, order := range ordersCopy {
		// 取得當前市價
		currentPrice, ok := GlobalPriceCache.GetPrice(order.Symbol)
		if !ok {
			continue
		}

		shouldExecute := false

		// 判斷是否應該執行訂單
		if order.Side == models.OrderSideBuy {
			// 買入限價單：當市價 <= 限價時執行
			if currentPrice <= order.LimitPrice {
				shouldExecute = true
			}
		} else {
			// 賣出限價單：當市價 >= 限價時執行
			if currentPrice >= order.LimitPrice {
				shouldExecute = true
			}
		}

		if shouldExecute {
			log.Printf("Executing limit order #%d: %s %s at limit price %.2f, current price %.2f",
				order.Id, order.Side, order.Symbol, order.LimitPrice, currentPrice)

			// 執行限價單
			err := m.executeLimitOrder(order, currentPrice)
			if err != nil {
				log.Printf("Failed to execute limit order #%d: %v", order.Id, err)
			} else {
				// 從待處理列表中移除
				m.mu.Lock()
				delete(m.pendingOrders, order.Id)
				m.mu.Unlock()
			}
		}
	}
}

// ExecuteLimitOrder 執行限價單
func (m *LimitOrderMatcher) ExecuteLimitOrder(order *models.Order, currentPrice float64) error {
	return m.executeLimitOrder(order, currentPrice)
}

// executeLimitOrder 執行限價單
func (m *LimitOrderMatcher) executeLimitOrder(order *models.Order, currentPrice float64) error {
	// 解析交易對
	base, quote, err := models.ParseSymbol(order.Symbol)
	if err != nil {
		return err
	}

	// 開始資料庫交易
	o := orm.NewOrm()
	to, err := o.Begin()
	if err != nil {
		models.UpdateOrderStatus(orm.NewOrm(), order.Id, models.OrderStatusFailed, 0, 0, "Failed to start transaction")
		return fmt.Errorf("failed to start transaction: %v", err)
	}

	shouldRollback := true
	defer func() {
		if shouldRollback {
			to.Rollback()
			if err != nil {
				models.UpdateOrderStatus(orm.NewOrm(), order.Id, models.OrderStatusFailed, 0, 0, err.Error())
			}
		}
	}()

	// 執行交易邏輯
	var totalAmount float64
	var actualQuantity float64

	// 取得 User ID（需要先讀取完整的 order 資料）
	fullOrder := &models.Order{Id: order.Id}
	if err := orm.NewOrm().Read(fullOrder); err != nil {
		return fmt.Errorf("failed to read order: %v", err)
	}

	// 需要載入 User 關聯
	orm.NewOrm().LoadRelated(fullOrder, "User")
	userId := fullOrder.User.Id

	// 區分槓桿訂單和現貨訂單的執行邏輯
	if fullOrder.IsLeverageOrder {
		// 槓桿訂單：不扣除完整 USDT，只更新幣種錢包
		actualQuantity = fullOrder.Quantity
		totalAmount = fullOrder.Quantity * fullOrder.LimitPrice

		if fullOrder.Side == models.OrderSideBuy {
			// 買入：增加 base 幣錢包，不扣除 USDT
			baseWallet := &models.Wallet{}
			err = to.QueryTable(new(models.Wallet)).
				Filter("User__Id", userId).
				Filter("Symbol", base).
				One(baseWallet)

			if err == orm.ErrNoRows {
				// 錢包不存在，建立一個
				baseWallet = &models.Wallet{
					User:    &models.User{Id: userId},
					Symbol:  base,
					Balance: 0,
					Locked:  0,
				}
				_, err = to.Insert(baseWallet)
				if err != nil {
					return fmt.Errorf("failed to create %s wallet: %v", base, err)
				}
			} else if err != nil {
				return fmt.Errorf("failed to get %s wallet: %v", base, err)
			}

			// 增加 base 幣餘額
			baseWallet.Balance += actualQuantity
			_, err = to.Update(baseWallet, "Balance")
			if err != nil {
				return fmt.Errorf("failed to update %s balance: %v", base, err)
			}
		} else {
			// 賣出：減少 base 幣錢包，增加 USDT
			baseWallet := &models.Wallet{}
			err = to.QueryTable(new(models.Wallet)).
				Filter("User__Id", userId).
				Filter("Symbol", base).
				One(baseWallet)

			if err != nil {
				return fmt.Errorf("failed to get %s wallet: %v", base, err)
			}

			if baseWallet.GetAvailableBalance() < actualQuantity {
				return errors.New("insufficient coin balance")
			}

			// 減少 base 幣餘額
			baseWallet.Balance -= actualQuantity
			_, err = to.Update(baseWallet, "Balance")
			if err != nil {
				return fmt.Errorf("failed to update %s balance: %v", base, err)
			}

			// 增加 USDT 錢包（不扣除保證金）
			quoteWallet := &models.Wallet{}
			err = to.QueryTable(new(models.Wallet)).
				Filter("User__Id", userId).
				Filter("Symbol", quote).
				One(quoteWallet)

			if err != nil {
				return fmt.Errorf("failed to get %s wallet: %v", quote, err)
			}

			quoteWallet.Balance += totalAmount
			_, err = to.Update(quoteWallet, "Balance")
			if err != nil {
				return fmt.Errorf("failed to update %s balance: %v", quote, err)
			}
		}
	} else {
		// 現貨訂單：正常執行，扣除完整 USDT
		if fullOrder.Side == models.OrderSideBuy {
			// 買入：計算需要的 USDT 金額 = 幣種數量 × 限價
			usdtAmount := fullOrder.Quantity * fullOrder.LimitPrice
			totalAmount, actualQuantity, err = executeBuyOrder(to, userId, base, quote, usdtAmount, fullOrder.LimitPrice, fullOrder.Id)
		} else {
			// 賣出：直接使用幣種數量
			totalAmount, actualQuantity, err = executeSellOrder(to, userId, base, quote, fullOrder.Quantity, fullOrder.LimitPrice, fullOrder.Id)
		}

		if err != nil {
			return err
		}
	}

	// 更新訂單狀態
	err = models.UpdateOrderStatus(to, fullOrder.Id, models.OrderStatusCompleted, currentPrice, totalAmount, "")
	if err != nil {
		return fmt.Errorf("failed to update order: %v", err)
	}

	// 提交交易
	err = to.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	shouldRollback = false

	log.Printf("Limit order #%d executed successfully: %s %s %.8f at price %.2f, total %.2f",
		order.Id, order.Side, order.Symbol, actualQuantity, currentPrice, totalAmount)

	// 發送 WebSocket 通知給用戶
	message := models.NewLimitOrderFilledMessage(
		fullOrder.Id,
		fullOrder.Symbol,
		fullOrder.Side,
		fullOrder.LimitPrice,
		currentPrice,
		actualQuantity,
		totalAmount,
	)
	hub.GlobalHub.BroadcastToUser(userId, message.ToJSON())

	// 如果這是一個槓桿訂單，建立槓桿倉位
	if fullOrder.IsLeverageOrder {
		positionSide := models.PositionSide(fullOrder.PositionSideStr)
		// fullOrder.Quantity 代表想要購買的幣種數量
		// 保證金 = (數量 × 限價) / 槓桿倍數
		margin := (fullOrder.Quantity * fullOrder.LimitPrice) / float64(fullOrder.Leverage)

		// 計算爆倉價格
		var liquidationPrice float64
		liquidationRatio := 0.9 / float64(fullOrder.Leverage)
		if positionSide == models.PositionSideLong {
			liquidationPrice = fullOrder.LimitPrice * (1 - liquidationRatio)
		} else {
			liquidationPrice = fullOrder.LimitPrice * (1 + liquidationRatio)
		}

		position := &models.LeveragePosition{
			User:             &models.User{Id: userId},
			Order:            fullOrder,
			Symbol:           fullOrder.Symbol,
			Side:             positionSide,
			Leverage:         fullOrder.Leverage,
			EntryPrice:       fullOrder.LimitPrice,
			Quantity:         actualQuantity, // 實際購買的幣種數量
			Margin:           margin,
			LiquidationPrice: liquidationPrice,
			UnrealizedPnL:    0,
			RealizedPnL:      0,
			Status:           models.PositionStatusOpen,
		}

		// 保存槓桿倉位
		o := orm.NewOrm()
		_, err := o.Insert(position)
		if err != nil {
			log.Printf("Warning: Failed to create leverage position for order #%d: %v", fullOrder.Id, err)
		} else {
			log.Printf("Leverage position #%d created: User=%d, Symbol=%s, Side=%s, Leverage=%dx, Quantity=%.8f, EntryPrice=%.2f, Margin=%.2f",
				position.Id, userId, fullOrder.Symbol, positionSide, fullOrder.Leverage, fullOrder.Quantity, fullOrder.LimitPrice, margin)

			// 從 USDT 錢包扣除保證金
			wallet, err := models.GetWalletByUserAndSymbol(userId, "USDT")
			if err == nil {
				wallet.Balance -= margin
				wallet.Locked += margin
				_, err = o.Update(wallet, "Balance", "Locked")
				if err != nil {
					log.Printf("Warning: Failed to deduct margin from user %d: %v", userId, err)
				}
			}

			// 發送槓桿倉位開倉通知給用戶
			posMessage := models.NewLeveragePositionOpenedMessage(position)
			hub.GlobalHub.BroadcastToUser(userId, posMessage.ToJSON())
		}
	}

	return nil
}

// AddOrder 新增限價單到監控列表
func (m *LimitOrderMatcher) AddOrder(order *models.Order) {
	if order.Type != models.OrderTypeLimit || order.Status != models.OrderStatusPending {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.pendingOrders[order.Id] = order
	log.Printf("Added limit order #%d to matcher: %s %s at %.2f", order.Id, order.Side, order.Symbol, order.LimitPrice)
}

// RemoveOrder 從監控列表移除訂單（取消時使用）
func (m *LimitOrderMatcher) RemoveOrder(orderId int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.pendingOrders, orderId)
	log.Printf("Removed order #%d from matcher", orderId)
}

// PlaceLimitOrder 下限價單
func PlaceLimitOrder(userId int64, symbol string, side models.OrderSide, quantity float64, limitPrice float64) (*models.Order, error) {
	// 1. 驗證輸入
	if quantity <= 0 {
		return nil, errors.New("quantity must be positive")
	}

	if limitPrice <= 0 {
		return nil, errors.New("limit price must be positive")
	}

	_, _, err := models.ParseSymbol(symbol)
	if err != nil {
		return nil, err
	}

	// 2. 建立限價單
	order, err := models.CreateOrder(userId, symbol, models.OrderTypeLimit, side, quantity, &limitPrice)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %v", err)
	}

	// 3. 獲取當前市價（用於日誌記錄）
	currentPrice, _ := GlobalPriceCache.GetPrice(symbol)

	// 4. 所有限價單都直接加入 matcher，讓 matcher 統一管理執行時機
	// 不在下單時檢查是否應該立即成交，因為：
	// 1. 避免並發問題（多個地方同時檢查和執行）
	// 2. matcher 會定期檢查所有待處理的限價單，確保不會遺漏
	// 3. 這樣能保證限價單的執行順序和一致性

	log.Printf("Limit order #%d added to matcher: %s %s at %.2f (current price: %.2f)",
		order.Id, side, symbol, limitPrice, currentPrice)
	GlobalLimitOrderMatcher.AddOrder(order)

	return order, nil
}
