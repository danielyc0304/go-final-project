package main

import (
	_ "backend/db"
	"backend/hub"
	_ "backend/routers"
	"backend/services"
	"backend/utils"

	beego "github.com/beego/beego/v2/server/web"
)

func init() {
	hub.GlobalHub = hub.NewHub()
	go hub.GlobalHub.Run()
	go services.ConnectToBinance(hub.GlobalHub)

	// 啟動限價單撮合服務
	services.GlobalLimitOrderMatcher.Start()
}

func main() {
	if beego.BConfig.RunMode == "dev" {
		beego.BConfig.WebConfig.DirectoryIndex = true
		beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
	}

	// 設定 CORS 中間件
	beego.InsertFilter("*", beego.BeforeRouter, utils.CORSFilter)

	// 設定 JWT 驗證中間件
	beego.InsertFilter("*", beego.BeforeExec, utils.AuthMiddleware)

	beego.Run()
}
