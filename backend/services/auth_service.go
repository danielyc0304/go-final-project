package services

import (
	"backend/models"
	"backend/utils"
	"errors"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

func Registration(m models.Registration) (id int64, err error) {
	if _, err = models.GetUserByEmail(m.Email); err != nil && err != orm.ErrNoRows {
		return
	} else if err == nil {
		err = errors.New("user already registered")
		return
	}

	var hashedPassword string
	if hashedPassword, err = utils.HashPassword(m.Password); err != nil {
		return
	}
	user := models.User{
		Name:     m.Name,
		Email:    m.Email,
		Password: hashedPassword,
	}
	if id, err = models.AddUser(&user); err != nil {
		return
	}

	// 為新用戶初始化默認錢包
	if err = models.InitializeDefaultWallets(id); err != nil {
		return
	}

	return
}

func Login(m models.Login) (token string, expiredAt time.Time, err error) {
	var user *models.User
	if user, err = models.GetUserByEmail(m.Email); err != nil && err != orm.ErrNoRows {
		return
	} else if err != nil {
		err = errors.New("invalid email or password")
		return
	}

	if err = utils.CheckPassword(user.Password, m.Password); err != nil {
		return
	}

	ttl := 30 * time.Minute
	if token, err = utils.GenerateToken(user.Id, ttl); err != nil {
		return
	}
	expiredAt = time.Now().Add(ttl)
	return
}
