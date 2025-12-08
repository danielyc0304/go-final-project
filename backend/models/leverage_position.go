package models

import (
	"errors"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// PositionSide 倉位方向
type PositionSide string

const (
	PositionSideLong  PositionSide = "LONG"  // 做多（看漲）
	PositionSideShort PositionSide = "SHORT" // 做空（看跌）
)

// PositionStatus 倉位狀態
type PositionStatus string

const (
	PositionStatusOpen       PositionStatus = "OPEN"       // 持倉中
	PositionStatusClosed     PositionStatus = "CLOSED"     // 已平倉
	PositionStatusLiquidated PositionStatus = "LIQUIDATED" // 已爆倉
)

// LeveragePosition 槓桿倉位
type LeveragePosition struct {
	Id               int64          `orm:"auto" json:"id"`
	User             *User          `orm:"rel(fk)" json:"-"`
	Order            *Order         `orm:"rel(fk);null" json:"-"`                                  // 關聯的開倉訂單
	Symbol           string         `orm:"size(20)" json:"symbol"`                                 // 交易對：BTCUSDT, ETHUSDT, SOLUSDT
	Side             PositionSide   `orm:"size(10)" json:"side"`                                   // LONG or SHORT
	Leverage         int            `orm:"default(1)" json:"leverage"`                             // 槓桿倍數：1-10
	EntryPrice       float64        `orm:"digits(20);decimals(8)" json:"entryPrice"`               // 開倉價格
	Quantity         float64        `orm:"digits(20);decimals(8)" json:"quantity"`                 // 持倉數量
	Margin           float64        `orm:"digits(20);decimals(8)" json:"margin"`                   // 保證金（USDT）
	LiquidationPrice float64        `orm:"digits(20);decimals(8)" json:"liquidationPrice"`         // 爆倉價格
	UnrealizedPnL    float64        `orm:"digits(20);decimals(8)" json:"unrealizedPnl"`            // 未實現盈虧
	RealizedPnL      float64        `orm:"digits(20);decimals(8)" json:"realizedPnl"`              // 已實現盈虧
	ExitPrice        float64        `orm:"digits(20);decimals(8);null" json:"exitPrice,omitempty"` // 平倉價格
	Status           PositionStatus `orm:"size(20)" json:"status"`
	CreatedAt        time.Time      `orm:"auto_now_add;type(datetime)" json:"createdAt"`
	UpdatedAt        time.Time      `orm:"auto_now;type(datetime)" json:"updatedAt"`
	ClosedAt         *time.Time     `orm:"type(datetime);null" json:"closedAt,omitempty"`
}

func init() {
	orm.RegisterModel(new(LeveragePosition))
}

// TableName 指定資料表名稱
func (l *LeveragePosition) TableName() string {
	return "leverage_position"
}

// CalculateLiquidationPrice 計算爆倉價格
func (l *LeveragePosition) CalculateLiquidationPrice() float64 {
	// 爆倉價格計算：
	// 做多：爆倉價 = 開倉價 * (1 - 0.9 / 槓桿)
	// 做空：爆倉價 = 開倉價 * (1 + 0.9 / 槓桿)
	// 0.9 是維持保證金率（90%），留 10% 緩衝

	liquidationRatio := 0.9 / float64(l.Leverage)

	if l.Side == PositionSideLong {
		return l.EntryPrice * (1 - liquidationRatio)
	} else {
		return l.EntryPrice * (1 + liquidationRatio)
	}
}

// CalculateUnrealizedPnL 計算未實現盈虧
func (l *LeveragePosition) CalculateUnrealizedPnL(currentPrice float64) float64 {
	if l.Side == PositionSideLong {
		// 做多：(當前價 - 開倉價) * 數量
		return (currentPrice - l.EntryPrice) * l.Quantity
	} else {
		// 做空：(開倉價 - 當前價) * 數量
		return (l.EntryPrice - currentPrice) * l.Quantity
	}
}

// IsLiquidated 檢查是否應該爆倉
func (l *LeveragePosition) IsLiquidated(currentPrice float64) bool {
	if l.Side == PositionSideLong {
		// 做多：當前價 <= 爆倉價
		return currentPrice <= l.LiquidationPrice
	} else {
		// 做空：當前價 >= 爆倉價
		return currentPrice >= l.LiquidationPrice
	}
}

// CreateLeveragePosition 創建槓桿倉位
func CreateLeveragePosition(userId int64, symbol string, side PositionSide, leverage int, entryPrice float64, quantity float64, margin float64) (*LeveragePosition, error) {
	o := orm.NewOrm()

	// 驗證槓桿倍數
	if leverage < 1 || leverage > 100 {
		return nil, errors.New("leverage must be between 1 and 100")
	}

	position := &LeveragePosition{
		User:       &User{Id: userId},
		Symbol:     symbol,
		Side:       side,
		Leverage:   leverage,
		EntryPrice: entryPrice,
		Quantity:   quantity,
		Margin:     margin,
		Status:     PositionStatusOpen,
	}

	// 計算爆倉價格
	position.LiquidationPrice = position.CalculateLiquidationPrice()

	_, err := o.Insert(position)
	if err != nil {
		return nil, err
	}

	return position, nil
}

// GetOpenPositionsByUser 查詢使用者的持倉
func GetOpenPositionsByUser(userId int64) ([]*LeveragePosition, error) {
	o := orm.NewOrm()
	var positions []*LeveragePosition
	_, err := o.QueryTable(new(LeveragePosition)).
		Filter("User__Id", userId).
		Filter("Status", PositionStatusOpen).
		RelatedSel().
		OrderBy("-CreatedAt").
		All(&positions)
	return positions, err
}

// GetAllOpenPositions 查詢所有持倉（用於爆倉檢查）
func GetAllOpenPositions() ([]*LeveragePosition, error) {
	o := orm.NewOrm()
	var positions []*LeveragePosition
	_, err := o.QueryTable(new(LeveragePosition)).
		Filter("Status", PositionStatusOpen).
		RelatedSel().
		All(&positions)
	return positions, err
}

// GetPositionById 根據 ID 查詢倉位
func GetPositionById(id int64) (*LeveragePosition, error) {
	o := orm.NewOrm()
	position := &LeveragePosition{Id: id}
	err := o.Read(position)
	if err == orm.ErrNoRows {
		return nil, errors.New("position not found")
	}
	return position, err
}

// ClosePosition 平倉
func ClosePosition(positionId int64, userId int64, exitPrice float64) error {
	o := orm.NewOrm()

	position, err := GetPositionById(positionId)
	if err != nil {
		return err
	}

	// 載入 User 關聯
	o.LoadRelated(position, "User")

	// 驗證所有權
	if position.User.Id != userId {
		return errors.New("unauthorized: position does not belong to user")
	}

	// 驗證狀態
	if position.Status != PositionStatusOpen {
		return errors.New("position is not open")
	}

	// 計算已實現盈虧
	position.RealizedPnL = position.CalculateUnrealizedPnL(exitPrice)
	position.ExitPrice = exitPrice
	position.Status = PositionStatusClosed
	now := time.Now()
	position.ClosedAt = &now

	_, err = o.Update(position)
	return err
}

// LiquidatePosition 強制平倉（爆倉）
func LiquidatePosition(positionId int64) error {
	o := orm.NewOrm()

	position, err := GetPositionById(positionId)
	if err != nil {
		return err
	}

	if position.Status != PositionStatusOpen {
		return errors.New("position is not open")
	}

	// 爆倉時虧損全部保證金
	position.RealizedPnL = -position.Margin
	position.ExitPrice = position.LiquidationPrice
	position.Status = PositionStatusLiquidated
	now := time.Now()
	position.ClosedAt = &now

	_, err = o.Update(position)
	return err
}

// UpdatePositionPnL 更新倉位盈虧
func UpdatePositionPnL(position *LeveragePosition, currentPrice float64) error {
	o := orm.NewOrm()
	position.UnrealizedPnL = position.CalculateUnrealizedPnL(currentPrice)
	_, err := o.Update(position, "UnrealizedPnL", "UpdatedAt")
	return err
}

// GetPositionsByUserAndSymbol 查詢特定交易對的倉位
func GetPositionsByUserAndSymbol(userId int64, symbol string, limit int, offset int) ([]*LeveragePosition, error) {
	o := orm.NewOrm()
	var positions []*LeveragePosition
	_, err := o.QueryTable(new(LeveragePosition)).
		Filter("User__Id", userId).
		Filter("Symbol", symbol).
		RelatedSel().
		OrderBy("-CreatedAt").
		Limit(limit, offset).
		All(&positions)
	return positions, err
}

// GetAllPositionsByUser 查詢使用者所有倉位（包含已平倉）
func GetAllPositionsByUser(userId int64, limit int, offset int) ([]*LeveragePosition, error) {
	o := orm.NewOrm()
	var positions []*LeveragePosition
	_, err := o.QueryTable(new(LeveragePosition)).
		Filter("User__Id", userId).
		RelatedSel().
		OrderBy("-CreatedAt").
		Limit(limit, offset).
		All(&positions)
	return positions, err
}

// CalculateRequiredMargin 計算所需保證金
func CalculateRequiredMargin(entryPrice float64, quantity float64, leverage int) float64 {
	// 保證金 = (開倉價 * 數量) / 槓桿
	return (entryPrice * quantity) / float64(leverage)
}

// CalculatePositionValue 計算倉位價值
func CalculatePositionValue(price float64, quantity float64) float64 {
	return price * quantity
}

// GetPositionPnLPercentage 計算盈虧百分比
func GetPositionPnLPercentage(position *LeveragePosition) float64 {
	if position.Margin == 0 {
		return 0
	}
	return (position.UnrealizedPnL / position.Margin) * 100
}
