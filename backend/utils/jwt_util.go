package utils

import (
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func getJwtSecret() (s string) {
	s = os.Getenv("JWT_SECRET")
	if s == "" {
		s = "1a9c7205a64fac856e71d90da0d1324541e0995eaf89e9d0e4f2c39491170454"
	}
	return
}

func GenerateToken(userID int64, ttl time.Duration) (token string, err error) {
	now := time.Now()
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer:    "Quantis",
			Subject:   strconv.FormatInt(userID, 10),
		},
	}

	unsignedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err = unsignedToken.SignedString([]byte(getJwtSecret()))
	return
}

func ParseToken(token string) (claims *Claims, err error) {
	var parsedToken *jwt.Token
	if parsedToken, err = jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (secret any, err error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			err = jwt.ErrTokenUnverifiable
			return
		}
		secret = []byte(getJwtSecret())
		return
	}); err != nil {
		return
	}

	var ok bool
	if claims, ok = parsedToken.Claims.(*Claims); ok && parsedToken.Valid {
		return
	}
	err = jwt.ErrTokenInvalidClaims
	return
}
