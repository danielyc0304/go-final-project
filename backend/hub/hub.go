// 檔案：backend/hub/hub.go
package hub

import (
	"log"

	"github.com/gorilla/websocket"
)

// Client 結構體代表一個前端連線
// Conn 是 WebSocket 連線本身
// Send 是一個 channel，用來傳送訊息給這個客戶端
type Client struct {
	Conn *websocket.Conn
	Send chan []byte
}

// Hub 負責管理所有的客戶端和廣播
type Hub struct {
	// 儲存所有已註冊的客戶端
	Clients map[*Client]bool

	// 廣播 channel。當有訊息放入這個 channel，Hub 會將它廣播給所有客戶端
	Broadcast chan []byte

	// 註冊 channel。當有新客戶端連線時，會放入這裡
	Register chan *Client

	// 註銷 channel。當有客戶端斷線時，會放入這裡
	Unregister chan *Client
}

// NewHub 是一個工廠方法，用來建立一個新的 Hub
func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

// Run 啟動 Hub 的核心邏輯，它會不斷監聽四個 channel
// 這個函式必須在一個獨立的 goroutine 中執行
func (h *Hub) Run() {
	// 使用無限迴圈
	for {
		// select 會等待 channel 收到訊息
		select {
		case client := <-h.Register:
			// 處理新客戶端註冊
			h.Clients[client] = true
			log.Printf("Client registered. Total clients: %d", len(h.Clients))

		case client := <-h.Unregister:
			// 處理客戶端註銷
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client) // 從 map 中刪除
				close(client.Send)        // 關閉它的 Send channel
				log.Printf("Client unregistered. Total clients: %d", len(h.Clients))
			}

		case message := <-h.Broadcast:
			// 處理廣播訊息
			// 遍歷所有已註冊的客戶端
			for client := range h.Clients {
				// 將訊息放入客戶端的 Send channel
				select {
				case client.Send <- message:
					// 成功放入
				default:
					// 如果 Send channel 已滿（可能客戶端處理太慢或已斷線）
					// 則關閉這個客戶端
					close(client.Send)
					delete(h.Clients, client)
					log.Println("Client send buffer full. Disconnecting client.")
				}
			}
		}
	}
}
