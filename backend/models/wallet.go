package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// Wallet 使用者的錢包，記錄各種貨幣的餘額
type Wallet struct {
	Id        int64     `orm:"auto" json:"id"`
	User      *User     `orm:"rel(fk)" json:"-"`
	Symbol    string    `orm:"size(20)" json:"symbol"`                // 幣種：USDT, BTC, ETH, SOL
	Balance   float64   `orm:"digits(20);decimals(8)" json:"balance"` // 餘額
	Locked    float64   `orm:"digits(20);decimals(8)" json:"locked"`  // 鎖定金額（掛單中）
	CreatedAt time.Time `orm:"auto_now_add;type(datetime)" json:"createdAt"`
	UpdatedAt time.Time `orm:"auto_now;type(datetime)" json:"updatedAt"`
}

func init() {
	orm.RegisterModel(new(Wallet))
}

// GetWalletByUserAndSymbol 根據使用者和幣種查詢錢包
func GetWalletByUserAndSymbol(userId int64, symbol string) (*Wallet, error) {
	o := orm.NewOrm()
	wallet := &Wallet{}
	err := o.QueryTable(new(Wallet)).
		Filter("User__Id", userId).
		Filter("Symbol", symbol).
		RelatedSel().
		One(wallet)

	if err == orm.ErrNoRows {
		return nil, errors.New("wallet not found")
	}
	return wallet, err
}

// GetAllWalletsByUser 查詢使用者所有錢包
func GetAllWalletsByUser(userId int64) ([]*Wallet, error) {
	o := orm.NewOrm()
	var wallets []*Wallet
	_, err := o.QueryTable(new(Wallet)).
		Filter("User__Id", userId).
		RelatedSel().
		All(&wallets)
	return wallets, err
}

// CreateWallet 建立新錢包
func CreateWallet(userId int64, symbol string, initialBalance float64) (*Wallet, error) {
	o := orm.NewOrm()

	// 檢查是否已存在
	existing := &Wallet{}
	err := o.QueryTable(new(Wallet)).
		Filter("User__Id", userId).
		Filter("Symbol", symbol).
		One(existing)

	if err == nil {
		// 錢包已存在，直接返回而不報錯
		return existing, nil
	}

	wallet := &Wallet{
		User:    &User{Id: userId},
		Symbol:  symbol,
		Balance: initialBalance,
		Locked:  0,
	}

	id, err := o.Insert(wallet)
	if err != nil {
		return nil, err
	}
	wallet.Id = id
	return wallet, nil
}

// UpdateBalance 更新餘額（需要在交易中使用）
func UpdateBalance(o orm.QueryExecutor, walletId int64, balanceChange float64, lockedChange float64) error {
	wallet := &Wallet{Id: walletId}
	ormer := orm.NewOrm()
	if err := ormer.Read(wallet); err != nil {
		return err
	}

	newBalance := wallet.Balance + balanceChange
	newLocked := wallet.Locked + lockedChange

	if newBalance < 0 {
		return errors.New("insufficient balance")
	}
	if newLocked < 0 {
		return errors.New("invalid locked amount")
	}

	wallet.Balance = newBalance
	wallet.Locked = newLocked

	_, err := o.Update(wallet, "Balance", "Locked")
	return err
}

// GetAvailableBalance 取得可用餘額
func (w *Wallet) GetAvailableBalance() float64 {
	return w.Balance - w.Locked
}

// InitializeDefaultWallets 為新使用者初始化預設錢包
func InitializeDefaultWallets(userId int64) error {
	symbols := []string{"USDT", "BTC", "ETH", "SOL"}
	initialBalances := map[string]float64{
		"USDT": 100000.0, // 初始給 10萬 USDT
		"BTC":  0.0,
		"ETH":  0.0,
		"SOL":  0.0,
	}

	for _, symbol := range symbols {
		_, err := CreateWallet(userId, symbol, initialBalances[symbol])
		if err != nil {
			return fmt.Errorf("failed to create wallet for %s: %v", symbol, err)
		}
	}
	return nil
}
