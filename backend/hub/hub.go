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
	Conn   *websocket.Conn
	Send   chan []byte
	UserId int64 // 用戶 ID，用於路由特定用戶的消息
}

// Hub 負責管理所有的客戶端和廣播
type Hub struct {
	Clients         map[*Client]bool    // 所有客戶端
	ClientsByUserId map[int64][]*Client // 按 userId 分組的客戶端
	Broadcast       chan []byte         // 廣播消息
	UserBroadcast   chan UserMessage    // 按用戶廣播消息
	Register        chan *Client
	Unregister      chan *Client
}

// UserMessage 用戶特定的消息
type UserMessage struct {
	UserId  int64  // 接收消息的用戶 ID
	Message []byte // 消息內容
}

// NewHub 建立一個新的 Hub
func NewHub() *Hub {
	return &Hub{
		Clients:         make(map[*Client]bool),
		ClientsByUserId: make(map[int64][]*Client),
		Broadcast:       make(chan []byte, 256),
		UserBroadcast:   make(chan UserMessage, 256),
		Register:        make(chan *Client),
		Unregister:      make(chan *Client),
	}
}

// Run 啟動 Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
			if client.UserId > 0 {
				h.ClientsByUserId[client.UserId] = append(h.ClientsByUserId[client.UserId], client)
				log.Printf("Client registered for user %d. Total clients: %d", client.UserId, len(h.Clients))
			}
		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				// 從 ClientsByUserId 中移除
				if client.UserId > 0 {
					clients := h.ClientsByUserId[client.UserId]
					for i, c := range clients {
						if c == client {
							h.ClientsByUserId[client.UserId] = append(clients[:i], clients[i+1:]...)
							break
						}
					}
					if len(h.ClientsByUserId[client.UserId]) == 0 {
						delete(h.ClientsByUserId, client.UserId)
					}
				}
				close(client.Send)
				log.Printf("Client unregistered. Total clients: %d", len(h.Clients))
			}
		case message := <-h.Broadcast:
			// 廣播給所有客戶端
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client)
					log.Println("Client send buffer full. Disconnecting client.")
				}
			}
		case userMsg := <-h.UserBroadcast:
			// 廣播給特定用戶的所有客戶端
			if clients, ok := h.ClientsByUserId[userMsg.UserId]; ok {
				for _, client := range clients {
					select {
					case client.Send <- userMsg.Message:
					default:
						close(client.Send)
						delete(h.Clients, client)
						log.Println("Client send buffer full. Disconnecting client.")
					}
				}
			}
		}
	}
}

// BroadcastToUser 廣播消息給特定用戶
func (h *Hub) BroadcastToUser(userId int64, message []byte) {
	select {
	case h.UserBroadcast <- UserMessage{UserId: userId, Message: message}:
	default:
		log.Printf("User broadcast channel full, dropping message for user %d", userId)
	}
}
