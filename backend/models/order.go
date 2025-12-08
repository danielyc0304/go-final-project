package models

import (
	"errors"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// OrderSide 買入或賣出
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// OrderType 訂單類型
type OrderType string

const (
	OrderTypeMarket OrderType = "MARKET" // 市價單
	OrderTypeLimit  OrderType = "LIMIT"  // 限價單
)

// OrderStatus 訂單狀態
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"   // 待處理（限價單等待執行）
	OrderStatusCompleted OrderStatus = "COMPLETED" // 已完成
	OrderStatusFailed    OrderStatus = "FAILED"    // 失敗
	OrderStatusCanceled  OrderStatus = "CANCELED"  // 已取消
)

// Order 訂單
type Order struct {
	Id              int64       `orm:"auto" json:"id"`
	User            *User       `orm:"rel(fk)" json:"-"`
	Symbol          string      `orm:"size(20)" json:"symbol"`                                  // 交易對：BTCUSDT, ETHUSDT, SOLUSDT
	Type            OrderType   `orm:"size(10)" json:"type"`                                    // MARKET or LIMIT
	Side            OrderSide   `orm:"size(10)" json:"side"`                                    // BUY or SELL
	Quantity        float64     `orm:"digits(20);decimals(8)" json:"quantity"`                  // 交易數量
	LimitPrice      float64     `orm:"digits(20);decimals(8);null" json:"limitPrice,omitempty"` // 限價（僅限價單使用）
	Price           float64     `orm:"digits(20);decimals(8)" json:"price"`                     // 成交價格
	TotalAmount     float64     `orm:"digits(20);decimals(8)" json:"totalAmount"`               // 總金額
	IsLeverageOrder bool        `orm:"default(false)" json:"isLeverageOrder"`                   // 是否是槓桿訂單
	Leverage        int         `orm:"default(1);null" json:"leverage,omitempty"`               // 槓桿倍數（僅槓桿訂單使用）
	PositionSideStr string      `orm:"size(10);null" json:"positionSide,omitempty"`             // 倉位方向：LONG or SHORT（僅槓桿訂單使用）
	Status          OrderStatus `orm:"size(20)" json:"status"`
	ErrorMsg        string      `orm:"size(500);null" json:"errorMsg,omitempty"`
	CreatedAt       time.Time   `orm:"auto_now_add;type(datetime)" json:"createdAt"`
	UpdatedAt       time.Time   `orm:"auto_now;type(datetime)" json:"updatedAt"`
}

func init() {
	orm.RegisterModel(new(Order))
}

// CreateOrder 建立新訂單
func CreateOrder(userId int64, symbol string, orderType OrderType, side OrderSide, quantity float64, limitPrice *float64) (*Order, error) {
	o := orm.NewOrm()

	order := &Order{
		User:     &User{Id: userId},
		Symbol:   symbol,
		Type:     orderType,
		Side:     side,
		Quantity: quantity,
		Status:   OrderStatusPending,
	}

	if limitPrice != nil {
		order.LimitPrice = *limitPrice
	}

	id, err := o.Insert(order)
	if err != nil {
		return nil, err
	}
	order.Id = id
	return order, nil
}

// CreateLeverageOrder 建立槓桿訂單
func CreateLeverageOrder(userId int64, symbol string, orderType OrderType, side OrderSide, quantity float64, limitPrice *float64, leverage int, positionSide PositionSide) (*Order, error) {
	o := orm.NewOrm()

	order := &Order{
		User:            &User{Id: userId},
		Symbol:          symbol,
		Type:            orderType,
		Side:            side,
		Quantity:        quantity,
		Status:          OrderStatusPending,
		IsLeverageOrder: true,
		Leverage:        leverage,
		PositionSideStr: string(positionSide),
	}

	if limitPrice != nil {
		order.LimitPrice = *limitPrice
	}

	id, err := o.Insert(order)
	if err != nil {
		return nil, err
	}
	order.Id = id
	return order, nil
}

// UpdateOrderStatus 更新訂單狀態
func UpdateOrderStatus(tx orm.QueryExecutor, orderId int64, status OrderStatus, price float64, totalAmount float64, errorMsg string) error {
	order := &Order{Id: orderId}
	ormer := orm.NewOrm()
	if err := ormer.Read(order); err != nil {
		return err
	}

	order.Status = status
	order.Price = price
	order.TotalAmount = totalAmount
	order.ErrorMsg = errorMsg

	_, err := tx.Update(order, "Status", "Price", "TotalAmount", "ErrorMsg")
	return err
}

// GetOrderById 根據 ID 查詢訂單
func GetOrderById(orderId int64) (*Order, error) {
	o := orm.NewOrm()
	order := &Order{Id: orderId}
	if err := o.Read(order); err != nil {
		return nil, err
	}
	return order, nil
}

// GetOrdersByUser 查詢使用者的所有訂單
func GetOrdersByUser(userId int64, limit int, offset int) ([]*Order, error) {
	o := orm.NewOrm()
	var orders []*Order
	_, err := o.QueryTable(new(Order)).
		Filter("User__Id", userId).
		OrderBy("-CreatedAt").
		Limit(limit, offset).
		All(&orders)
	return orders, err
}

// GetOrdersByUserAndSymbol 查詢使用者某個交易對的訂單
func GetOrdersByUserAndSymbol(userId int64, symbol string, limit int, offset int) ([]*Order, error) {
	o := orm.NewOrm()
	var orders []*Order
	_, err := o.QueryTable(new(Order)).
		Filter("User__Id", userId).
		Filter("Symbol", symbol).
		OrderBy("-CreatedAt").
		Limit(limit, offset).
		All(&orders)
	return orders, err
}

// GetPendingLimitOrders 查詢所有待執行的限價單
func GetPendingLimitOrders() ([]*Order, error) {
	o := orm.NewOrm()
	var orders []*Order
	_, err := o.QueryTable(new(Order)).
		Filter("Type", OrderTypeLimit).
		Filter("Status", OrderStatusPending).
		RelatedSel().
		All(&orders)
	return orders, err
}

// CancelOrder 取消訂單
func CancelOrder(orderId int64, userId int64) error {
	o := orm.NewOrm()
	order := &Order{Id: orderId}

	if err := o.Read(order); err != nil {
		return err
	}

	// 檢查訂單所有權
	if order.User.Id != userId {
		return errors.New("unauthorized: order does not belong to user")
	}

	// 只能取消待處理的訂單
	if order.Status != OrderStatusPending {
		return errors.New("order cannot be canceled")
	}

	order.Status = OrderStatusCanceled
	_, err := o.Update(order, "Status")
	return err
}

// ParseSymbol 解析交易對，返回 base 和 quote 幣種
// 例如：BTCUSDT -> (BTC, USDT)
func ParseSymbol(symbol string) (base string, quote string, err error) {
	validSymbols := map[string][2]string{
		"BTCUSDT": {"BTC", "USDT"},
		"ETHUSDT": {"ETH", "USDT"},
		"SOLUSDT": {"SOL", "USDT"},
	}

	if pair, ok := validSymbols[symbol]; ok {
		return pair[0], pair[1], nil
	}
	return "", "", errors.New("invalid trading symbol")
}
