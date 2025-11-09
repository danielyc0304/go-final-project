// 檔案：backend/hub/hub.go (修正版)
package hub

import (
	"log"

	"github.com/gorilla/websocket"
)

// GlobalHub 是我們全局的 Hub 實例，
// 所有的 goroutine 都會來存取它
var GlobalHub *Hub

// Client 結構體代表一個前端連線
type Client struct {
	Conn *websocket.Conn
	Send chan []byte
}

// Hub 負責管理所有的客戶端和廣播
type Hub struct {
	Clients    map[*Client]bool
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
}

// NewHub 建立一個新的 Hub
func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

// Run 啟動 Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
			log.Printf("Client registered. Total clients: %d", len(h.Clients))
		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
				log.Printf("Client unregistered. Total clients: %d", len(h.Clients))
			}
		case message := <-h.Broadcast:
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client)
					log.Println("Client send buffer full. Disconnecting client.")
				}
			}
		}
	}
}
