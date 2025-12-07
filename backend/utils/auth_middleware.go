package utils

import (
	"strings"

	"github.com/beego/beego/v2/server/web/context"
)

// AuthMiddleware JWT 驗證中間件
// 該中間件檢查 Authorization header 中的 JWT token
func AuthMiddleware(ctx *context.Context) {
	// 不需要驗證的路由（白名單）
	unprotectedRoutes := []string{
		"/v1/auth/registration",
		"/v1/auth/login",
		"/v1/market/klines",
	}

	// 檢查路由是否在白名單中
	path := ctx.Request.URL.Path
	isUnprotected := false
	for _, route := range unprotectedRoutes {
		if strings.HasPrefix(path, route) {
			isUnprotected = true
			break
		}
	}

	// WebSocket 路由不需要驗證
	if strings.HasPrefix(path, "/ws") {
		return
	}

	if isUnprotected {
		return
	}

	// 驗證 JWT Token
	authHeader := ctx.Input.Header("Authorization")
	if authHeader == "" {
		ctx.Output.SetStatus(401)
		ctx.Output.JSON(map[string]interface{}{
			"success": false,
			"error":   "missing authorization header",
		}, false, false)
		return
	}

	// 移除 "Bearer " 前綴
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		ctx.Output.SetStatus(401)
		ctx.Output.JSON(map[string]interface{}{
			"success": false,
			"error":   "invalid authorization header format",
		}, false, false)
		return
	}

	// 解析 token
	claims, err := ParseToken(tokenString)
	if err != nil {
		ctx.Output.SetStatus(401)
		ctx.Output.JSON(map[string]interface{}{
			"success": false,
			"error":   "invalid or expired token: " + err.Error(),
		}, false, false)
		return
	}

	// 將 userID 存到 context 中，方便後續使用
	ctx.Input.SetData("userID", claims.UserID)
}
