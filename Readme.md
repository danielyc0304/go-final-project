# 💰 虛擬貨幣模擬交易平台 (Crypto Trading Simulator)

## 🚀 專案簡介

本專案旨在提供一個虛擬貨幣的模擬交易環境。
使用者可以在前端介面上進行買賣操作，所有交易邏輯、帳戶餘額和歷史紀錄皆由高效的 Go 語言後端處理和儲存。我們使用 **Docker Compose** 確保開發環境的快速一致性。

* **前端:** Vite (提供熱重載開發環境，運行於主機)
* **後端:** Go 語言 (負責交易邏輯、API 處理，運行於 Docker 容器)

## 🛠️ 開發環境啟動

請確保您的系統已安裝 **Docker Desktop** 和 **Git**。

### 方式 1: 啟動所有服務
在專案根目錄下，執行以下命令。這將建置並啟動專案中所有容器(包含前後端、資料庫)。

```bash
docker compose up --build
```

若您想背景執行，可以在專案根目錄下，執行以下命令。
```bash
docker compose up --build -d
```
---
### 方式 2: 分別啟動不同容器
#### 步驟 1: 啟動後端服務 (Go)

在專案根目錄下，執行以下命令。這將建置並啟動 Go 後端容器，並掛載程式碼 Volume 以便您修改。

```bash
docker compose up --build -d backend
```

> **💡 說明:** 這裡的 `backend` 服務使用 `Dockerfile.backend` 中的 `builder` 階段映像檔。

#### 步驟 2: 啟動前端開發伺服器 (Vite)

為了最佳的熱重載性能 (HMR)，請在主機上運行 Vite 開發伺服器。

```bash
# 進入前端目錄
cd frontend/

# 安裝依賴 (如果尚未安裝)
npm install

# 啟動 Vite Dev Server
npm run dev
```

成功啟動後，前端介面將運行於 `http://localhost:5173`。

## 🌐 服務與埠號 (Port) 定義

所有服務的埠號映射都定義在 `docker-compose.yml` 中。

| 服務名稱 | 容器埠號 | 主機埠號 | 說明 |
| :--- | :--- | :--- | :--- |
| **前端 (Vite Dev)** | `5173` | `5173` | **開發環境** 熱重載埠號 |
| **後端 (Go)** | `3000` | `3000` | 後端 API 服務埠號 |

### API 通訊路徑

由於前端運行在主機，後端埠號被映射到主機的 `3000`，因此前端應透過以下路徑呼叫 API：

```
http://localhost:3000/[您的API路徑]
```

-----

## 📝 Git Commit 訊息規範

為了維持清晰、可追溯的專案歷史記錄，我們採用以下常見的 Commit 類型分類：

| 類型 (Type) | 簡介 | 範例 |
| :--- | :--- | :--- |
| **feat** | 新增功能 (Feature) | `feat: 新增買入交易 API 端點` |
| **fix** | 錯誤修復 (Bug Fix) | `fix: 修正餘額計算時的浮點數精度錯誤` |
| **docs** | 文件變更 | `docs: 更新 README 中的啟動步驟` |
| **style** | 程式碼格式調整 (不影響邏輯) | `style: 調整 Go 程式碼的命名慣例` |
| **refactor** | 程式碼重構 (不新增功能或修復錯誤) | `refactor: 優化服務層的交易處理邏輯` |
| **test** | 測試相關的變動 (新增、修改測試) | `test: 增加單元測試覆蓋率` |
| **chore** | 維護或其他不影響運行的變動 | `chore: 更新 Dockerfile 的基礎映像版本` |