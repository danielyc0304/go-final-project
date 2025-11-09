// 檔案：backend/controllers/websocket.go (修正版)
package controllers

import (
	"backend/hub" // 匯入 hub package
	"log"
	"net/http"

	"github.com/beego/beego/v2/server/web"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// writePump 將 Hub 來的訊息寫入 WebSocket 連線
func writePump(c *hub.Client) {
	defer func() {
		c.Conn.Close()
	}()
	for {
		message, ok := <-c.Send
		if !ok {
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}

// readPump 從 WebSocket 連線讀取訊息 (用來偵測斷線)
func readPump(c *hub.Client) {
	defer func() {
		hub.GlobalHub.Unregister <- c // 從 Hub 註銷
		c.Conn.Close()
	}()

	// 保持連線，但忽略所有來自前端的訊息
	for {
		if _, _, err := c.Conn.ReadMessage(); err != nil {
			break // 連線關閉或出錯
		}
		// 訊息被忽略
	}
}

type WebSocketController struct {
	web.Controller
}

// Get() 處理 /ws 的連線請求
func (wsc *WebSocketController) Get() {
	conn, err := upgrader.Upgrade(wsc.Ctx.ResponseWriter, wsc.Ctx.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		wsc.Abort("500")
		return
	}
	log.Println("Client connected to WebSocket...")

	client := &hub.Client{
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	// 向 GlobalHub 註冊這個 Client
	hub.GlobalHub.Register <- client

	// 啟動 writePump 和 readPump
	go writePump(client)
	go readPump(client)
}
