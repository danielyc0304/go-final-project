import React, { useEffect, useMemo, useRef, useState } from "react";
import "./App.css";
import Welcome from "./Welcome.jsx";

/* ===== 小工具 ===== */
const fmt = new Intl.NumberFormat("zh-TW", { maximumFractionDigits: 2 });
const fmt4 = new Intl.NumberFormat("zh-TW", { maximumFractionDigits: 6 });

/* 以現價為中心產生 N 檔：BID 高→低、ASK 低→高 (這部分仍維持模擬，因為幣安沒接 OrderBook) */
function makeOrderBookAround(mid, levels = 6) {
  const price = Number(mid) || 0;
  if (price === 0) return { bids: [], asks: [] };

  const step = Math.max(0.01, price * 0.0001); // 縮小價差讓看起來真實點
  const bestBid = price - step;
  const bestAsk = price + step;

  const bids = Array.from({ length: levels }, (_, i) => ({
    price: bestBid - i * step,
    qty: Number((Math.random() * 0.5 + 0.01).toFixed(4)),
  }));

  const asks = Array.from({ length: levels }, (_, i) => ({
    price: bestAsk + i * step,
    qty: Number((Math.random() * 0.5 + 0.01).toFixed(4)),
  }));

  return { bids, asks };
}

/* 可搜尋商品列表 */
const SYMBOLS = [
  { symbol: "BTCUSDT", name: "Bitcoin" },
  { symbol: "ETHUSDT", name: "Ethereum" },
  { symbol: "SOLUSDT", name: "Solana" },
  { symbol: "BNBUSDT", name: "BNB" },
  { symbol: "XRPUSDT", name: "XRP" },
  { symbol: "ADAUSDT", name: "Cardano" },
  { symbol: "DOGEUSDT", name: "Dogecoin" },
];

/* ===== K 線元件 (修復資料載入時序問題) ===== */
function CandlestickChart({ data }) {
  const containerRef = useRef(null);
  const chartRef = useRef(null);
  const seriesRef = useRef(null);
  const prevDataLengthRef = useRef(0);
  
  // [新增] 使用 ref 隨時記錄最新的 data，解決閉包舊資料問題
  const latestDataRef = useRef(data);
  useEffect(() => {
    latestDataRef.current = data;
  }, [data]);

  // 1. 初始化圖表
  useEffect(() => {
    let chart;
    let series;
    let ro;

    (async () => {
      const { createChart, CandlestickSeries } = await import("lightweight-charts");
      const el = containerRef.current;
      if (!el) return;

      chart = createChart(el, {
        width: el.clientWidth,
        height: 560,
        layout: { background: { type: "solid", color: "#0f1115" }, textColor: "#e6e6e6" },
        grid: { vertLines: { color: "#1b1f2a" }, horzLines: { color: "#1b1f2a" } },
        rightPriceScale: { borderColor: "#2a2f3b" },
        timeScale: { borderColor: "#2a2f3b", timeVisible: true, secondsVisible: false },
        crosshair: { mode: 1 },
      });

      series = chart.addSeries(CandlestickSeries, {
        upColor: "#26a69a", downColor: "#ef5350",
        wickUpColor: "#26a69a", wickDownColor: "#ef5350",
        borderVisible: false,
      });

      // [關鍵修正] 初始化完成時，直接讀取 ref 裡的「最新資料」，而不是閉包裡的舊 data
      if (latestDataRef.current && latestDataRef.current.length > 0) {
        series.setData(latestDataRef.current);
        prevDataLengthRef.current = latestDataRef.current.length;
      }

      chartRef.current = chart;
      seriesRef.current = series;

      ro = new ResizeObserver(() => {
        if (containerRef.current) {
          chart.applyOptions({ width: containerRef.current.clientWidth });
        }
      });
      ro.observe(el);
    })();

    return () => {
      if (ro) ro.disconnect();
      if (chart) chart.remove();
    };
  }, []); // 只執行一次

  // 2. 數據更新邏輯
  useEffect(() => {
    // 如果圖表還沒建立好，就先略過，反正初始化那邊(上面)會去抓最新的
    if (!seriesRef.current) return;
    
    // 如果資料是空的，也沒必要畫
    if (data.length === 0) return;

    const prevLength = prevDataLengthRef.current;
    const currLength = data.length;
    const lastCandle = data[currLength - 1];

    // 判斷是歷史載入(大量) 還是 即時更新(單筆)
    if (prevLength === 0 || Math.abs(currLength - prevLength) > 1) {
      seriesRef.current.setData(data);
    } else {
      seriesRef.current.update(lastCandle);
    }

    prevDataLengthRef.current = currLength;
  }, [data]);

  return <div className="chart" ref={containerRef} />;
}

/* ===== 主應用 ===== */
export default function App() {
  const [logged, setLogged] = useState(false);
  const [symbol, setSymbol] = useState("BTCUSDT");
  
  // [修改點 1] 改為空陣列，等待 API 填入
  const [kData, setKData] = useState([]); 
  
  // 計算現價 (取最後一根 K 線的收盤價)
  const lastPrice = kData.length > 0 ? kData[kData.length - 1].close : 0;

  const [orderBook, setOrderBook] = useState({ bids: [], asks: [] });

  // 現價漲跌顏色邏輯
  const prevRef = useRef(lastPrice);
  const priceTrend = lastPrice > prevRef.current ? "up" : lastPrice < prevRef.current ? "down" : "";
  useEffect(() => { prevRef.current = lastPrice; }, [lastPrice]);


  // ------------------------------------------------------------
  // [修改點 2] 載入歷史資料 (登入後或切換幣種時)
  // ------------------------------------------------------------
  useEffect(() => {
    if (!logged) return;

    const fetchHistory = async () => {
      try {
        setKData([]); // 切換前先清空，避免圖表殘留
        
        // 呼叫後端 API (透過 Vite Proxy 轉發 /v1 -> backend:8080)
        const res = await fetch(`/v1/market/klines?symbol=${symbol}&interval=1m&limit=1000`);
        const json = await res.json();
        
        if (json.success && Array.isArray(json.data)) {
          // 確保時間由舊到新排序
          const sorted = json.data.sort((a, b) => a.time - b.time);
          setKData(sorted);
        }
      } catch (err) {
        console.error("無法取得歷史資料:", err);
      }
    };

    fetchHistory();
  }, [logged, symbol]);

  // ------------------------------------------------------------
  // [修改點 3] WebSocket 即時更新
  // ------------------------------------------------------------
  useEffect(() => {
    if (!logged) return;

    // 建立 WS 連線 (透過 Vite Proxy 轉發 /ws -> backend:8080)
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`; 
    const ws = new WebSocket(wsUrl);

    ws.onopen = () => console.log("WebSocket 已連線");

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        
        // 1. [除錯] 確保有收到資料
        // console.log("收到 WS:", msg); 

        const s = symbol.toLowerCase();
        
        // 2. 確保是當前幣種的 trade 數據
        if (msg.stream === `${s}@trade` && msg.data) {
          const price = parseFloat(msg.data.p);
          const time = Math.floor(msg.data.T / 1000); 

          // 3. [除錯] 確保有進入更新邏輯
          // console.log("更新價格:", price);

          setKData((prev) => {
            if (prev.length === 0) return prev;

            const newData = [...prev];
            const lastIndex = newData.length - 1;
            const last = newData[lastIndex];

            // 判斷是否為新的一分鐘
            if (time >= last.time + 60) {
                // 開新 K 線
                const newTime = Math.floor(time / 60) * 60;
                newData.push({
                    time: newTime,
                    open: price, high: price, low: price, close: price
                });
                if (newData.length > 2000) newData.shift();
            } else {
                // [修正點] 建立一個"新物件"來更新，確保 React 偵測到變化
                newData[lastIndex] = {
                    ...last,
                    close: price,
                    high: Math.max(last.high, price),
                    low: Math.min(last.low, price)
                };
            }
            return newData;
          });
        }
      } catch (e) {
        console.error("WS Error:", e);
      }
    };

    return () => ws.close();
  }, [logged, symbol]);
  // ------------------------------------------------------------

  // 訂單簿連動 (維持模擬)
  useEffect(() => {
    if (lastPrice > 0) setOrderBook(makeOrderBookAround(lastPrice, 6));
  }, [lastPrice]);

  // 搜尋
  const [q, setQ] = useState("");
  const suggestions = useMemo(() => {
    if (!q.trim()) return [];
    return SYMBOLS.filter(x => x.symbol.toLowerCase().includes(q.toLowerCase())).slice(0, 8);
  }, [q]);

  // 資金與倉位
  const [cash, setCash] = useState(100000);
  const [positions, setPositions] = useState([]);
  const [realized, setRealized] = useState(0);

  // 下單
  function submitOrder({ side, price, orderType, qtyCoin, leverage, notional, margin }) {
    if (cash < margin) return alert("餘額不足");
    setCash(c => c - margin);
    const id = `${Date.now()}`;
    const closePrice = side === "BUY" ? price * (1 - 1 / leverage) : price * (1 + 1 / leverage);
    
    setPositions(ps => [{
      id, symbol, side, qty: qtyCoin, entry: price, leverage, notional, margin, closePrice, orderType
    }, ...ps]);
  }

  // 平倉
  function closePosition(pid) {
    const p = positions.find(x => x.id === pid);
    if (!p) return;
    const pnl = (lastPrice - p.entry) * p.qty * (p.side === "BUY" ? 1 : -1);
    setCash(c => c + p.margin + pnl);
    setRealized(r => r + pnl);
    setPositions(ps => ps.filter(x => x.id !== pid));
  }

  // 計算損益
  const unreal = useMemo(() => positions.reduce((sum, p) => sum + (lastPrice - p.entry) * p.qty * (p.side === "BUY" ? 1 : -1), 0), [positions, lastPrice]);
  const totalVal = useMemo(() => positions.reduce((sum, p) => sum + (p.qty || 0) * lastPrice, 0), [positions, lastPrice]);

  // if (!logged) return <LoginPage onLogin={() => setLogged(true)} />;
  if (!logged) return <Welcome setLogged = {setLogged}/>;

  return (
    <div className="app">
      <header className="header">
        <div className="brand">Quantis</div>
        <div className="header-right">USDT: {fmt.format(cash)}</div>
      </header>

      <main className="content">
        <div className="search-wrap">
          <input className="search" value={q} onChange={e => setQ(e.target.value)} placeholder="搜尋 (例如 BTC, ETH)..." />
          {suggestions.length > 0 && (
            <div className="suggest">
              {suggestions.map(s => (
                <div key={s.symbol} className="suggest-item" onClick={() => { setSymbol(s.symbol); setQ(""); }}>
                  <span className="sym">{s.symbol}</span> <span className="nm">{s.name}</span>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="trade-3col">
          <section className="chart-wrap">
            <div className="chart-title">{symbol}</div>
            {/* 有資料才畫圖，避免報錯 */}
            {kData.length > 0 ? (
                 <CandlestickChart data={kData} /> 
            ) : (
                <div style={{height: 560, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#666'}}>
                    正在連線至 Binance 取得數據...
                </div>
            )}
          </section>

          <OrderBookPanel symbol={symbol} lastPrice={lastPrice} trend={priceTrend} orderBook={orderBook} />
          <TradePanel symbol={symbol} lastPrice={lastPrice} onSubmit={submitOrder} />
        </div>

        <PositionsTable positions={positions} markPrice={lastPrice} closePosition={closePosition} totalPositionValue={totalVal} unreal={unreal} realized={realized} />
      </main>
    </div>
  );
}

/* ===== 子元件 ===== */

// function LoginPage({ onLogin }) {
//   const [u, setU] = useState("");
//   const [p, setP] = useState("");
//   return (
//     <div className="login-wrap">
//       <div className="login-card">
//         <h1>Quantis 模擬交易</h1>
//         <p className="sub">系統已連接 Binance 真實行情</p>
//         <label>帳號 (任意)</label>
//         <input value={u} onChange={(e) => setU(e.target.value)} placeholder="輸入帳號" />
//         <label>密碼 (任意)</label>
//         <input type="password" value={p} onChange={(e) => setP(e.target.value)} placeholder="輸入密碼" />
//         <button className="primary" onClick={onLogin} style={{marginTop: 20}}>
//           登入系統
//         </button>
//       </div>
//     </div>
//   );
// }

function OrderBookPanel({ symbol, lastPrice, trend, orderBook }) {
  const chg = (Math.random() - 0.5) * 2.3; // 模擬 24h 漲跌
  return (
    <section className="quote-wrap">
      <div className="panel">
        <div className="panel-head"><div className="panel-title">{symbol}</div></div>
        <div className="quote-price">
          現價 <span className={`price-value ${trend === "up" ? "price-up" : trend === "down" ? "price-down" : ""}`}>{fmt.format(lastPrice)}</span> USDT
        </div>
        
        <div className="kv">
            <div><span>24h 漲跌</span><b className={chg >= 0 ? "up" : "down"}>{chg >= 0 ? "+" : ""}{chg.toFixed(2)}%</b></div>
            <div><span>24h 最高</span><b>{fmt.format(lastPrice * 1.02)}</b></div>
            <div><span>24h 最低</span><b>{fmt.format(lastPrice * 0.98)}</b></div>
            <div><span>成交量(估)</span><b>{fmt4.format(3000 + Math.random() * 800)}</b></div>
        </div>

        <div className="orderbook">
          <div className="ob-cols">
            <div>
              <div className="ob-hint">買進(BID)</div>
              {orderBook.bids.map((r, i) => (
                <div className="ob-row bid" key={`b-${i}`}><span className="price">{fmt.format(r.price)}</span><span className="qty">{fmt4.format(r.qty)}</span></div>
              ))}
            </div>
            <div>
              <div className="ob-hint">賣出(ASK)</div>
              {orderBook.asks.map((r, i) => (
                <div className="ob-row ask" key={`a-${i}`}><span className="price">{fmt.format(r.price)}</span><span className="qty">{fmt4.format(r.qty)}</span></div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

function TradePanel({ symbol, lastPrice, onSubmit }) {
    const [side, setSide] = useState("BUY");
    const [orderType, setOrderType] = useState("LIMIT");
    const [price, setPrice] = useState("");
    const [inputMode, setInputMode] = useState("COIN");
    const [qty, setQty] = useState("");
    const [lev, setLev] = useState(10);
    const [tpOn, setTpOn] = useState(false);
    const [tp, setTp] = useState("");
    const [slOn, setSlOn] = useState(false);
    const [sl, setSl] = useState("");

    // Limit 模式下自動填入現價
    useEffect(() => {
        if (orderType === "LIMIT" && lastPrice > 0) setPrice(lastPrice.toFixed(2));
    }, [orderType, lastPrice]);

    const parsedPrice = Number(price) || 0;
    const parsedQty = Number(qty) || 0;
    const basePrice = orderType === "MARKET" ? lastPrice : parsedPrice;

    let margin = 0, notional = 0, coinQty = 0;
    if (basePrice > 0 && parsedQty > 0 && lev > 0) {
        if (inputMode === "USD") {
            margin = parsedQty; notional = margin * lev; coinQty = notional / basePrice;
        } else {
            coinQty = parsedQty; notional = coinQty * basePrice; margin = notional / lev;
        }
    }
    const closePrice = basePrice > 0 && lev > 0 ? (side === "BUY" ? basePrice * (1 - 1 / lev) : basePrice * (1 + 1 / lev)) : 0;

    return (
        <section className="trade-wrap">
            <div className="panel">
                <div className="panel-head"><div className="panel-title">下單</div></div>
                <div className="side-switch">
                    <button className={`tab ${side === "BUY" ? "act" : ""}`} onClick={() => setSide("BUY")}>買進</button>
                    <button className={`tab ${side === "SELL" ? "act" : ""}`} onClick={() => setSide("SELL")}>賣出</button>
                </div>
                <div className="order-type-row">
                    <span className="order-type-label">下單方式</span>
                    <div className="order-type-switch">
                        <button className={`mini-tab ${orderType === "LIMIT" ? "act" : ""}`} onClick={() => setOrderType("LIMIT")}>限價</button>
                        <button className={`mini-tab ${orderType === "MARKET" ? "act" : ""}`} onClick={() => setOrderType("MARKET")}>市價</button>
                    </div>
                </div>

                <label>價格 (USDT)</label>
                <input value={orderType === "MARKET" ? "" : price} onChange={e => setPrice(e.target.value)} disabled={orderType === "MARKET"} placeholder={orderType === "MARKET" ? `${fmt.format(lastPrice)}（市價）` : ""} />

                <div className="qty-row">
                    <label>{inputMode === "COIN" ? `數量 (${symbol.replace("USDT", "")})` : "保證金 (USDT)"}</label>
                    <button className="mini-switch" onClick={() => setInputMode(m => m === "COIN" ? "USD" : "COIN")}>切換為 {inputMode === "COIN" ? "USDT" : "幣數"}</button>
                </div>
                <input value={qty} onChange={e => setQty(e.target.value)} placeholder={inputMode === "COIN" ? "例如 0.01" : "例如 100"} />

                <label>槓桿：{lev}x {closePrice ? <span className="lev-hint">（估平倉價：{fmt.format(closePrice)}）</span> : null}</label>
                <input type="range" min="1" max="50" value={lev} onChange={e => setLev(Number(e.target.value))} />
                
                <div className="tpsl-row">
                    <label className="inline"><input type="checkbox" checked={tpOn} onChange={e => setTpOn(e.target.checked)} /> TP</label>
                    <input disabled={!tpOn} value={tp} onChange={e => setTp(e.target.value)} placeholder="TP" />
                </div>
                <div className="tpsl-row">
                    <label className="inline"><input type="checkbox" checked={slOn} onChange={e => setSlOn(e.target.checked)} /> SL</label>
                    <input disabled={!slOn} value={sl} onChange={e => setSl(e.target.value)} placeholder="SL" />
                </div>

                <label>名目 (USDT)</label>
                <div className="display-box right">{fmt.format(notional)} USDT</div>

                <button className={`primary ${side === "SELL" ? "warn" : ""}`} onClick={() => {
                    if (!(basePrice > 0 && coinQty > 0 && notional > 0 && margin > 0)) return alert("請輸入有效價格、槓桿與數量");
                    onSubmit({ side, price: basePrice, orderType, qtyCoin: coinQty, leverage: lev, tpOn, tp, slOn, sl, notional, margin });
                }}>
                    送出{side === "BUY" ? "買進" : "賣出"}
                </button>
            </div>
        </section>
    );
}

function PositionsTable({ positions, markPrice, closePosition, totalPositionValue, unreal, realized }) {
    return (
        <section className="positions-wrap">
            <div className="asset-top mini">
                <div className="asset-title">倉位總額</div>
                <div className="asset-value">{fmt.format(totalPositionValue)} USDT</div>
                <div className="asset-sub">未實現損益：<span className={unreal >= 0 ? "up" : "down"}>{fmt.format(unreal)}</span></div>
                <div className="asset-sub">已實現損益：<span className={realized >= 0 ? "up" : "down"}>{fmt.format(realized)}</span></div>
            </div>
            <table className="tbl">
                <thead><tr><th>倉位</th><th>數量</th><th>名目價值</th><th>保證金</th><th>買入價</th><th>市場價</th><th>平倉價</th><th>未實現損益</th><th>TP</th><th>SL</th><th>操作</th></tr></thead>
                <tbody>
                    {positions.length === 0 && <tr><td colSpan="11" className="muted">尚無倉位</td></tr>}
                    {positions.map(p => {
                        const pnl = (markPrice - p.entry) * p.qty * (p.side === "BUY" ? 1 : -1);
                        return (
                            <tr key={p.id}>
                                <td>{p.symbol} <span className={p.side === "BUY" ? "up" : "down"}>{p.side === "BUY" ? "買進" : "賣出"}</span> {p.leverage}x <span className="order-type-tag">{p.orderType === "MARKET" ? "市價" : "限價"}</span></td>
                                <td>{fmt4.format(p.qty)}</td><td>{fmt.format(p.notional)}</td><td>{fmt.format(p.margin)}</td>
                                <td>{fmt.format(p.entry)}</td><td>{fmt.format(markPrice)}</td><td>{fmt.format(p.closePrice)}</td>
                                <td className={pnl >= 0 ? "up" : "down"}>{fmt.format(pnl)}</td>
                                <td>{p.tp ? fmt.format(p.tp) : "-"}</td><td>{p.sl ? fmt.format(p.sl) : "-"}</td>
                                <td><button className="ghost" onClick={() => closePosition(p.id)}>平倉</button></td>
                            </tr>
                        );
                    })}
                </tbody>
            </table>
        </section>
    );
}