package utils

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (hashedPassword string, err error) {
	var hashed []byte
	if hashed, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost); err != nil {
		return
	}
	hashedPassword = string(hashed)
	return
}

func CheckPassword(hashedPassword, password string) (err error) {
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return
}
