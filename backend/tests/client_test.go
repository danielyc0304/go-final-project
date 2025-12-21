package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"

	"github.com/gorilla/websocket"
)

// 設定區
const (
	Host      = "localhost:8080"
	Scheme    = "http"
	WSScheme  = "ws"
	JWT_TOKEN = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJpc3MiOiJRdWFudGlzIiwic3ViIjoiMSIsImV4cCI6MTc2NTIxNjE1NCwiaWF0IjoxNzY1MjE0MzU0fQ.yBHC2LXTrxtgMLt6tnsdId9kr4imoKDWQf_8vU2_UQg"
)

func main() {
	fmt.Println("🚀 開始系統測試...")
	fmt.Println("------------------------------------------------")
	testMarketAPI()
	fmt.Println("------------------------------------------------")
	testWebSocket()
}

// 測試 1: 呼叫後端 API 取得歷史資料
func testMarketAPI() {
	fmt.Println("📡 [Step 1] 測試 REST API: /v1/market/klines")

	apiURL := fmt.Sprintf("%s://%s/v1/market/klines?symbol=BTCUSDT&interval=1m&limit=5", Scheme, Host)
	fmt.Printf("   請求網址: %s\n", apiURL)

	resp, err := http.Get(apiURL)
	if err != nil {
		log.Fatalf("❌ API 請求失敗: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 200 {
		fmt.Println("✅ API 回應成功 (200 OK)")

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err == nil {
			if data, ok := result["data"].([]interface{}); ok {
				fmt.Printf("   收到資料筆數: %d 筆\n", len(data))
				if len(data) > 0 {
					fmt.Printf("   第一筆資料範例: %v\n", data[0])
				}
			}
		} else {
			fmt.Printf("⚠️ JSON 解析失敗: %v\n", err)
		}
	} else {
		fmt.Printf("❌ API 回應錯誤: Status %d\n   Body: %s\n", resp.StatusCode, string(body))
	}
}

// 測試 2: 連線 WebSocket 接收即時資料
func testWebSocket() {
	fmt.Println("🔌 [Step 2] 測試 WebSocket: /ws")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: WSScheme, Host: Host, Path: "/ws"}

	if JWT_TOKEN != "" {
		q := u.Query()
		q.Set("token", JWT_TOKEN)
		u.RawQuery = q.Encode()
	}
	fmt.Printf("   連線網址: %s\n", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("❌ WebSocket 連線失敗: %v\n   (請確認後端是否啟動，或 Token 是否正確)", err)
	}
	defer c.Close()

	fmt.Println("✅ WebSocket 連線成功！正在監聽訊息... (收到 10 筆後自動結束)")

	done := make(chan struct{})

	go func() {
		defer close(done)
		count := 0 // 計數器
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("⚠️ 讀取錯誤或連線關閉:", err)
				return
			}
			msgStr := string(message)
			if len(msgStr) > 100 {
				msgStr = msgStr[:100] + "..."
			}
			count++
			fmt.Printf("📩 [%d/10] 收到訊息: %s\n", count, msgStr)

			if count >= 10 {
				fmt.Println("🎉 測試完成！已成功接收 10 筆即時數據。")
				return
			}
		}
	}()

	// 等待中斷訊號 或 測試完成
	for {
		select {
		case <-done:
			c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		case <-interrupt:
			fmt.Println("\n🛑 測試結束，關閉連線。")
			c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		}
	}
}
