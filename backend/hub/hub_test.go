package hub

import (
	"testing"
	"time"
)

// TestNewHub 測試 Hub 的初始化
func TestNewHub(t *testing.T) {
	hub := NewHub()

	if hub == nil {
		t.Error("NewHub() returned nil")
	}

	if hub.Clients == nil {
		t.Error("Hub.Clients map is nil")
	}

	if hub.Broadcast == nil {
		t.Error("Hub.Broadcast channel is nil")
	}

	if cap(hub.Broadcast) != 1024 {
		t.Errorf("Expected Broadcast channel capacity 1024, got %d", cap(hub.Broadcast))
	}
}

// TestHubRun 測試 Hub 的運行與註冊邏輯
// 注意：這是一個並發測試
func TestHubRun(t *testing.T) {
	hub := NewHub()

	// 在背景啟動 Hub
	go hub.Run()

	// 模擬一個客戶端
	client := &Client{
		Send:   make(chan []byte, 256),
		UserId: 12345,
	}

	// 測試註冊 (Register)
	hub.Register <- client

	// 給一點時間讓 Hub 處理 (因為是並發的)
	time.Sleep(100 * time.Millisecond)

	// 驗證是否註冊成功
	if _, ok := hub.Clients[client]; !ok {
		t.Error("Client was not registered in hub.Clients")
	}

	if clients, ok := hub.ClientsByUserId[12345]; !ok || len(clients) == 0 {
		t.Error("Client was not registered in hub.ClientsByUserId")
	}

	// 測試註銷 (Unregister)
	hub.Unregister <- client
	time.Sleep(100 * time.Millisecond)

	if _, ok := hub.Clients[client]; ok {
		t.Error("Client was not unregistered from hub.Clients")
	}
}
