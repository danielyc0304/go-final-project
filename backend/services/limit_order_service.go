package services

import (
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
	lastCheckTime time.Time
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

	if order.Side == models.OrderSideBuy {
		// 買入：用 USDT 買入 base 幣
		totalAmount, actualQuantity, err = executeBuyOrder(to, userId, base, quote, order.Quantity, currentPrice, order.Id)
	} else {
		// 賣出：賣出 base 幣換 USDT
		totalAmount, actualQuantity, err = executeSellOrder(to, userId, base, quote, order.Quantity, currentPrice, order.Id)
	}

	if err != nil {
		return err
	}

	// 更新訂單狀態
	err = models.UpdateOrderStatus(to, order.Id, models.OrderStatusCompleted, currentPrice, totalAmount, "")
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

	// 3. 加入撮合器監控
	GlobalLimitOrderMatcher.AddOrder(order)

	log.Printf("Limit order created: User=%d, Symbol=%s, Side=%s, Quantity=%.8f, LimitPrice=%.2f",
		userId, symbol, side, quantity, limitPrice)

	return order, nil
}
