// 檔案：backend/controllers/websocket.go
package controllers

import (
	"backend/hub"  // 匯入我們剛建立的 hub
	"backend/main" // 匯入 main package，我們稍後會在這裡定義 GlobalHub
	"log"
	"net/http"

	"github.com/beego/beego/v2/server/web"
	"github.com/gorilla/websocket"
)

// 建立一個 "Upgrader"，它負責將 HTTP 連線升級為 WebSocket 連線
var upgrader = websocket.Upgrader{
	// 解決跨域問題：允許所有來源的連線 (在開發階段很方便)
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	// (可選) 設置讀寫緩衝區大小
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// 這兩個函式是 gorilla/websocket 的標準作法
// 負責處理「一個」客戶端的讀取和寫入

// writePump 將 Hub 來的訊息（c.Send channel）寫入 WebSocket 連線
func writePump(c *hub.Client) {
	// 確保函式結束時關閉連線
	defer func() {
		c.Conn.Close()
	}()
	for {
		// 從 Send channel 讀取訊息
		message, ok := <-c.Send
		if !ok {
			// 如果 channel 被 Hub 關閉了 (例如客戶端被註銷)
			// 傳送一個 CloseMessage 給前端，然後跳出迴圈
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		// 將訊息寫入連線
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			// 寫入失敗 (可能前端已斷線)
			return
		}
	}
}

// readPump 從 WebSocket 連線讀取訊息 (但目前我們不需要前端傳訊息)
func readPump(c *hub.Client, h *hub.Hub) {
	// 確保函式結束時，通知 Hub 註銷這個客戶端，並關閉連線
	defer func() {
		h.Unregister <- c // 傳送 unregister 訊號
		c.Conn.Close()
	}()

	// 你的任務是「廣播」，不是「接收」，所以我們忽略所有來自前端的訊息
	// 但我們需要這個迴圈來偵測前端是否「斷線」
	for {
		if _, _, err := c.Conn.ReadMessage(); err != nil {
			// 前端斷線 (err != nil)，跳出迴圈，defer 將會執行
			break
		}
		// 讀到的訊息被忽略
	}
}

// --- 你的 Beego Controller ---

type WebSocketController struct {
	web.Controller
}

// 當前端來訪問 /ws 時，Beego 會導向到這個 Get 方法
func (wsc *WebSocketController) Get() {
	// 1. 將 HTTP 連線升級為 WebSocket 連線
	conn, err := upgrader.Upgrade(wsc.Ctx.ResponseWriter, wsc.Ctx.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		// Beego 會自動處理錯誤回應
		wsc.Abort("500")
		return
	}
	// 注意：一旦升級成功，Beego 的 http-handler 就結束了。
	// 接下來的通訊完全由 conn (gorilla/websocket.Conn) 接管。
	log.Println("Client connected to WebSocket...")

	// 2. 建立一個新的 Client 實例
	client := &hub.Client{
		Conn: conn,
		Send: make(chan []byte, 256), // 建立一個帶有緩衝區的 Send channel
	}

	// 3. 向 GlobalHub 註冊這個 Client
	// 我們使用 main.GlobalHub (這是在步驟五才會定義的全局變數)
	main.GlobalHub.Register <- client

	// 4. 啟動 readPump 和 writePump
	// 必須在獨立的 goroutine 中執行，才不會卡住
	go writePump(client)
	go readPump(client, main.GlobalHub)
}
