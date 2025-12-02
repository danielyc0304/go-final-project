// 檔案：backend/services/binance_client.go
package services

import (
	"backend/hub" // 匯入 Hub，我們需要它的 Broadcast channel
	"log"
	"time" // 用於斷線重連

	"github.com/gorilla/websocket"
)

// 這是幣安 (Binance) 提供的 BTC/USDT 交易對的即時價格 WebSocket URL
const binanceURL = "wss://stream.binance.com:9443/stream?streams=btcusdt@trade/ethusdt@trade/solusdt@trade"

// ConnectToBinance 會開始連接幣安並將數據餵給 Hub
// 它應該在一個獨立的 goroutine 中執行
func ConnectToBinance(h *hub.Hub) {
	log.Println("Connecting to Binance WebSocket API...")

	// 使用無限迴圈，以便在斷線時自動重連
	for {
		// 1. 作為 "客戶端" 連線到幣安
		conn, _, err := websocket.DefaultDialer.Dial(binanceURL, nil)
		if err != nil {
			log.Println("Dial to Binance failed:", err, "Retrying in 5 seconds...")
			time.Sleep(5 * time.Second) // 等待 5 秒後重試
			continue                    // 重新執行迴Loop
		}

		log.Println("Successfully connected to Binance WebSocket API.")

		// 2. 在一個新迴圈中，不斷讀取幣安的訊息
		// (這是一個 "內迴圈")
		for {
			// 讀取來自幣安的訊息
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Read from Binance failed:", err)
				conn.Close() // 關閉舊連線
				break        // 跳出內迴圈，外迴圈將會執行並重連
			}

			// *** 核心中的核心 ***
			// 3. 將收到的 message 原封不動地廣播給所有前端客戶端
			//    我們把訊息餵給 Hub 的 Broadcast channel
			h.Broadcast <- message

			// 4. 同時更新價格快取，供交易系統使用
			GlobalPriceCache.UpdatePrice(message)

			// (可選) 如果你打開這個 log，你的終端機會被幣安的數據洗頻
			// log.Printf("Received from Binance: %s", message)
		}
	}
}
