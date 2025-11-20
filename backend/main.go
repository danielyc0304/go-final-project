package main

import (
	_ "backend/db"
	"backend/hub"
	_ "backend/routers"
	"backend/services"

	beego "github.com/beego/beego/v2/server/web"
)

func init() {
	hub.GlobalHub = hub.NewHub()
	go hub.GlobalHub.Run()
	go services.ConnectToBinance(hub.GlobalHub)
}

func main() {
	if beego.BConfig.RunMode == "dev" {
		beego.BConfig.WebConfig.DirectoryIndex = true
		beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
	}
	beego.Run()
}
