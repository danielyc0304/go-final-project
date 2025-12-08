package utils

import (
	"github.com/beego/beego/v2/server/web/context"
)

// CORSFilter 用於處理 CORS 請求
func CORSFilter(ctx *context.Context) {
	allowedOrigins := map[string]bool{
		"http://localhost:3000":      true,
		"http://localhost:5173":      true,
		"https://quantis.zzppss.org": true,
	}

	origin := ctx.Input.Header("Origin")
	if allowedOrigins[origin] {
		ctx.Output.Header("Access-Control-Allow-Origin", origin)
		ctx.Output.Header("Access-Control-Allow-Credentials", "true")
		ctx.Output.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		ctx.Output.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		ctx.Output.Header("Access-Control-Max-Age", "3600")

		// 處理 OPTIONS 預檢請求
		if ctx.Input.Method() == "OPTIONS" {
			ctx.Output.SetStatus(200)
			ctx.ResponseWriter.Write([]byte(""))
		}
	}
}
