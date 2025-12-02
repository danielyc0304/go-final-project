package services

import (
	"backend/models"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// PlaceMarketOrder 執行市價單交易
// symbol: 交易對（如 BTCUSDT）
// side: BUY 或 SELL
// quantity: 交易數量（對於 BUY 是指花費的 USDT 金額，對於 SELL 是指賣出的幣數量）
func PlaceMarketOrder(userId int64, symbol string, side models.OrderSide, quantity float64) (*models.Order, error) {
	// 1. 驗證輸入
	if quantity <= 0 {
		return nil, errors.New("quantity must be positive")
	}

	base, quote, err := models.ParseSymbol(symbol)
	if err != nil {
		return nil, err
	}

	// 2. 取得當前市價
	price, err := GlobalPriceCache.GetPriceWithTimeout(symbol, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to get market price: %v", err)
	}

	// 3. 建立訂單
	order, err := models.CreateOrder(userId, symbol, models.OrderTypeMarket, side, quantity, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %v", err)
	}

	// 4. 開始資料庫交易（確保原子性）
	o := orm.NewOrm()
	to, err := o.Begin()
	if err != nil {
		models.UpdateOrderStatus(orm.NewOrm(), order.Id, models.OrderStatusFailed, 0, 0, "Failed to start transaction")
		return nil, fmt.Errorf("failed to start transaction: %v", err)
	}

	// 用於追蹤交易是否需要回滾
	shouldRollback := true
	defer func() {
		if shouldRollback {
			to.Rollback()
			// 更新訂單狀態為失敗
			if err != nil {
				models.UpdateOrderStatus(orm.NewOrm(), order.Id, models.OrderStatusFailed, 0, 0, err.Error())
			}
		}
	}()

	// 5. 執行交易邏輯
	var totalAmount float64
	var actualQuantity float64

	if side == models.OrderSideBuy {
		// 買入：用 USDT 買入 base 幣
		totalAmount, actualQuantity, err = executeBuyOrder(to, userId, base, quote, quantity, price, order.Id)
	} else {
		// 賣出：賣出 base 幣換 USDT
		totalAmount, actualQuantity, err = executeSellOrder(to, userId, base, quote, quantity, price, order.Id)
	}

	if err != nil {
		return nil, err
	}

	// 6. 更新訂單狀態
	err = models.UpdateOrderStatus(to, order.Id, models.OrderStatusCompleted, price, totalAmount, "")
	if err != nil {
		return nil, fmt.Errorf("failed to update order: %v", err)
	}

	// 7. 提交交易
	err = to.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	// 標記交易成功，不需要回滾
	shouldRollback = false

	// 8. 重新讀取訂單以返回最新狀態
	order, _ = models.GetOrderById(order.Id)

	log.Printf("Order completed: User=%d, Symbol=%s, Side=%s, Quantity=%.8f, Price=%.2f, Total=%.2f",
		userId, symbol, side, actualQuantity, price, totalAmount)

	return order, nil
}

// executeBuyOrder 執行買入訂單
// quantity: 花費的 USDT 金額
func executeBuyOrder(tx orm.TxOrmer, userId int64, base string, quote string, usdtAmount float64, price float64, orderId int64) (totalAmount float64, actualQuantity float64, err error) {
	// 1. 檢查 USDT 餘額
	quoteWallet := &models.Wallet{}
	err = tx.QueryTable(new(models.Wallet)).
		Filter("User__Id", userId).
		Filter("Symbol", quote).
		One(quoteWallet)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get %s wallet: %v", quote, err)
	}

	if quoteWallet.GetAvailableBalance() < usdtAmount {
		return 0, 0, errors.New("insufficient USDT balance")
	}

	// 2. 計算能買到的幣數量
	actualQuantity = usdtAmount / price
	totalAmount = usdtAmount

	// 3. 取得或建立 base 幣錢包
	baseWallet := &models.Wallet{}
	err = tx.QueryTable(new(models.Wallet)).
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
		_, err = tx.Insert(baseWallet)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to create %s wallet: %v", base, err)
		}
	} else if err != nil {
		return 0, 0, fmt.Errorf("failed to get %s wallet: %v", base, err)
	}

	// 4. 更新 USDT 餘額（減少）
	quoteBalanceBefore := quoteWallet.Balance
	quoteWallet.Balance -= usdtAmount
	if quoteWallet.Balance < 0 {
		return 0, 0, errors.New("insufficient USDT balance")
	}
	_, err = tx.Update(quoteWallet, "Balance")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to update %s balance: %v", quote, err)
	}

	// 5. 更新 base 幣餘額（增加）
	baseBalanceBefore := baseWallet.Balance
	baseWallet.Balance += actualQuantity
	_, err = tx.Update(baseWallet, "Balance")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to update %s balance: %v", base, err)
	}

	// 6. 記錄交易（USDT 減少）
	quoteTx := &models.Transaction{
		User:          &models.User{Id: userId},
		Order:         &models.Order{Id: orderId},
		Type:          models.TransactionTypeBuy,
		Symbol:        quote,
		Amount:        -usdtAmount,
		BalanceBefore: quoteBalanceBefore,
		BalanceAfter:  quoteBalanceBefore - usdtAmount,
		Description:   fmt.Sprintf("Buy %s with %s at price %.2f", base, quote, price),
	}
	_, err = tx.Insert(quoteTx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create transaction: %v", err)
	}

	// 7. 記錄交易（base 幣增加）
	baseTx := &models.Transaction{
		User:          &models.User{Id: userId},
		Order:         &models.Order{Id: orderId},
		Type:          models.TransactionTypeBuy,
		Symbol:        base,
		Amount:        actualQuantity,
		BalanceBefore: baseBalanceBefore,
		BalanceAfter:  baseBalanceBefore + actualQuantity,
		Description:   fmt.Sprintf("Bought %.8f %s at price %.2f", actualQuantity, base, price),
	}
	_, err = tx.Insert(baseTx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create transaction: %v", err)
	}

	return totalAmount, actualQuantity, nil
}

// executeSellOrder 執行賣出訂單
// quantity: 賣出的 base 幣數量
func executeSellOrder(tx orm.TxOrmer, userId int64, base string, quote string, baseQuantity float64, price float64, orderId int64) (totalAmount float64, actualQuantity float64, err error) {
	// 1. 檢查 base 幣餘額
	baseWallet := &models.Wallet{}
	err = tx.QueryTable(new(models.Wallet)).
		Filter("User__Id", userId).
		Filter("Symbol", base).
		One(baseWallet)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get %s wallet: %v", base, err)
	}

	if baseWallet.GetAvailableBalance() < baseQuantity {
		return 0, 0, fmt.Errorf("insufficient %s balance", base)
	}

	// 2. 計算能得到的 USDT
	totalAmount = baseQuantity * price
	actualQuantity = baseQuantity

	// 3. 取得或建立 USDT 錢包
	quoteWallet := &models.Wallet{}
	err = tx.QueryTable(new(models.Wallet)).
		Filter("User__Id", userId).
		Filter("Symbol", quote).
		One(quoteWallet)

	if err == orm.ErrNoRows {
		// 錢包不存在，建立一個
		quoteWallet = &models.Wallet{
			User:    &models.User{Id: userId},
			Symbol:  quote,
			Balance: 0,
			Locked:  0,
		}
		_, err = tx.Insert(quoteWallet)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to create %s wallet: %v", quote, err)
		}
	} else if err != nil {
		return 0, 0, fmt.Errorf("failed to get %s wallet: %v", quote, err)
	}

	// 4. 更新 base 幣餘額（減少）
	baseBalanceBefore := baseWallet.Balance
	baseWallet.Balance -= baseQuantity
	if baseWallet.Balance < 0 {
		return 0, 0, fmt.Errorf("insufficient %s balance", base)
	}
	_, err = tx.Update(baseWallet, "Balance")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to update %s balance: %v", base, err)
	}

	// 5. 更新 USDT 餘額（增加）
	quoteBalanceBefore := quoteWallet.Balance
	quoteWallet.Balance += totalAmount
	_, err = tx.Update(quoteWallet, "Balance")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to update %s balance: %v", quote, err)
	}

	// 6. 記錄交易（base 幣減少）
	baseTx := &models.Transaction{
		User:          &models.User{Id: userId},
		Order:         &models.Order{Id: orderId},
		Type:          models.TransactionTypeSell,
		Symbol:        base,
		Amount:        -baseQuantity,
		BalanceBefore: baseBalanceBefore,
		BalanceAfter:  baseBalanceBefore - baseQuantity,
		Description:   fmt.Sprintf("Sell %s for %s at price %.2f", base, quote, price),
	}
	_, err = tx.Insert(baseTx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create transaction: %v", err)
	}

	// 7. 記錄交易（USDT 增加）
	quoteTx := &models.Transaction{
		User:          &models.User{Id: userId},
		Order:         &models.Order{Id: orderId},
		Type:          models.TransactionTypeSell,
		Symbol:        quote,
		Amount:        totalAmount,
		BalanceBefore: quoteBalanceBefore,
		BalanceAfter:  quoteBalanceBefore + totalAmount,
		Description:   fmt.Sprintf("Sold %.8f %s at price %.2f", baseQuantity, base, price),
	}
	_, err = tx.Insert(quoteTx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create transaction: %v", err)
	}

	return totalAmount, actualQuantity, nil
}
