package models

import (
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// TransactionType 交易類型
type TransactionType string

const (
	TransactionTypeBuy            TransactionType = "BUY"             // 買入
	TransactionTypeSell           TransactionType = "SELL"            // 賣出
	TransactionTypeDeposit        TransactionType = "DEPOSIT"         // 存款
	TransactionTypeWithdraw       TransactionType = "WITHDRAW"        // 提款
	TransactionTypeMarginDeposit  TransactionType = "MARGIN_DEPOSIT"  // 保證金存入
	TransactionTypeMarginWithdraw TransactionType = "MARGIN_WITHDRAW" // 保證金取出
	TransactionTypeLiquidation    TransactionType = "LIQUIDATION"     // 爆倉
)

// Transaction 交易記錄
type Transaction struct {
	Id            int64           `orm:"auto" json:"id"`
	User          *User           `orm:"rel(fk)" json:"-"`
	Order         *Order          `orm:"rel(fk);null" json:"-"` // 關聯訂單（可為空）
	Type          TransactionType `orm:"size(20)" json:"type"`
	Symbol        string          `orm:"size(20)" json:"symbol"`                      // 幣種
	Amount        float64         `orm:"digits(20);decimals(8)" json:"amount"`        // 金額（正數為增加，負數為減少）
	BalanceBefore float64         `orm:"digits(20);decimals(8)" json:"balanceBefore"` // 交易前餘額
	BalanceAfter  float64         `orm:"digits(20);decimals(8)" json:"balanceAfter"`  // 交易後餘額
	Description   string          `orm:"size(500);null" json:"description,omitempty"`
	CreatedAt     time.Time       `orm:"auto_now_add;type(datetime)" json:"createdAt"`
}

func init() {
	orm.RegisterModel(new(Transaction))
}

// CreateTransaction 建立交易記錄
func CreateTransaction(o orm.QueryExecutor, userId int64, orderId *int64, txType TransactionType, symbol string, amount float64, balanceBefore float64, balanceAfter float64, description string) (*Transaction, error) {
	tx := &Transaction{
		User:          &User{Id: userId},
		Type:          txType,
		Symbol:        symbol,
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		Description:   description,
	}

	if orderId != nil {
		tx.Order = &Order{Id: *orderId}
	}

	id, err := o.Insert(tx)
	if err != nil {
		return nil, err
	}
	tx.Id = id
	return tx, nil
}

// GetTransactionsByUser 查詢使用者的交易記錄
func GetTransactionsByUser(userId int64, limit int, offset int) ([]*Transaction, error) {
	o := orm.NewOrm()
	var transactions []*Transaction
	_, err := o.QueryTable(new(Transaction)).
		Filter("User__Id", userId).
		OrderBy("-CreatedAt").
		Limit(limit, offset).
		All(&transactions)
	return transactions, err
}

// GetTransactionsByUserAndSymbol 查詢使用者某個幣種的交易記錄
func GetTransactionsByUserAndSymbol(userId int64, symbol string, limit int, offset int) ([]*Transaction, error) {
	o := orm.NewOrm()
	var transactions []*Transaction
	_, err := o.QueryTable(new(Transaction)).
		Filter("User__Id", userId).
		Filter("Symbol", symbol).
		OrderBy("-CreatedAt").
		Limit(limit, offset).
		All(&transactions)
	return transactions, err
}

// GetTransactionsByOrder 查詢某個訂單的所有交易記錄
func GetTransactionsByOrder(orderId int64) ([]*Transaction, error) {
	o := orm.NewOrm()
	var transactions []*Transaction
	_, err := o.QueryTable(new(Transaction)).
		Filter("Order__Id", orderId).
		OrderBy("CreatedAt").
		All(&transactions)
	return transactions, err
}
