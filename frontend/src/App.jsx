import React, { useEffect, useMemo, useRef, useState } from "react";
import "./App.css";

/** ========= 小工具 ========= */
const fmt = new Intl.NumberFormat("zh-TW", { maximumFractionDigits: 2 });
const fmt4 = new Intl.NumberFormat("zh-TW", { maximumFractionDigits: 4 });
const nowSec = () => Math.floor(Date.now() / 1000);

function clamp(n, lo, hi) {
  return Math.min(Math.max(n, lo), hi);
}

/** 產生假資料：K 線 */
function genCandles(seedPrice = 105000, n = 180) {
  const out = [];
  let p = seedPrice;
  const t0 = nowSec() - n * 60;
  for (let i = 0; i < n; i++) {
    const time = t0 + i * 60; // 1 分鐘
    const drift = (Math.random() - 0.5) * (p * 0.0015);
    const open = p;
    let high = open + Math.abs(drift) * (1.2 + Math.random());
    let low = open - Math.abs(drift) * (1.2 + Math.random());
    let close = open + drift;
    const hi = Math.max(open, close, high);
    const lo = Math.min(open, close, low);
    out.push({
      time,
      open: Number(open.toFixed(2)),
      high: Number(hi.toFixed(2)),
      low: Number(lo.toFixed(2)),
      close: Number(close.toFixed(2)),
    });
    p = close;
  }
  return out;
}

/** 產生五檔（簡易） */
function makeOrderBook(mid) {
  const levels = 5;
  const step = Math.max(1, Math.round(mid * 0.0008));
  const bids = [];
  const asks = [];
  for (let i = levels; i >= 1; i--) {
    bids.push({
      price: mid - i * step,
      qty: Number((Math.random() * 0.8 + 0.2).toFixed(4)),
    });
  }
  for (let i = 1; i <= levels; i++) {
    asks.push({
      price: mid + i * step,
      qty: Number((Math.random() * 0.8 + 0.2).toFixed(4)),
    });
  }
  return { bids: bids.reverse(), asks };
}

/** 可搜尋清單（示意） */
const SYMBOLS = [
  { symbol: "BTCUSDT", name: "Bitcoin" },
  { symbol: "ETHUSDT", name: "Ethereum" },
  { symbol: "SOLUSDT", name: "Solana" },
  { symbol: "BNBUSDT", name: "BNB" },
  { symbol: "XRPUSDT", name: "XRP" },
  { symbol: "ADAUSDT", name: "Cardano" },
  { symbol: "DOGEUSDT", name: "Dogecoin" },
];

/** ========= K 線元件（lightweight-charts） ========= */
function CandlestickChart({ data }) {
  const ref = useRef(null);
  useEffect(() => {
    let chart, series, ro;
    let disposed = false;
    (async () => {
      const { createChart } = await import("lightweight-charts");
      if (disposed) return;
      chart = createChart(ref.current, {
        width: ref.current.clientWidth,
        height: 500,
        layout: { textColor: "#e6e6e6", background: { type: "solid", color: "#0f1115" }, watermark: { visible: false }, },
        crosshair: { mode: 1 },
        grid: {
          vertLines: { color: "#1b1f2a" },
          horzLines: { color: "#1b1f2a" },
        },
        rightPriceScale: { borderColor: "#2a2f3b" },
        timeScale: { borderColor: "#2a2f3b", timeVisible: true, secondsVisible: false },
      });
      series = chart.addCandlestickSeries({
        upColor: "#26a69a",
        downColor: "#ef5350",
        wickUpColor: "#26a69a",
        wickDownColor: "#ef5350",
        borderVisible: false,
      });
      series.setData(data);

      ro = new ResizeObserver(() => {
        chart.applyOptions({ width: ref.current.clientWidth });
      });
      ro.observe(ref.current);
    })();

    return () => {
      disposed = true;
      if (ro) ro.disconnect();
      if (ref.current && ref.current.firstChild) {
        ref.current.innerHTML = "";
      }
    };
  }, [data]);

  return <div className="chart" ref={ref} />;
}

/** ========= 主應用 ========= */
export default function App() {
  /** 登入與路由 */
  const [logged, setLogged] = useState(false);
  const [route, setRoute] = useState("home"); // home | assets
  const [drawerOpen, setDrawerOpen] = useState(false);

  /** 交易狀態 */
  const [symbol, setSymbol] = useState("BTCUSDT");
  const [kData, setKData] = useState(() => genCandles(105000));
  const lastPrice = kData[kData.length - 1]?.close ?? 0;
  const [orderBook, setOrderBook] = useState(() => makeOrderBook(lastPrice));
  const [showTradeForm, setShowTradeForm] = useState(false);

  /** 搜尋 */
  const [q, setQ] = useState("");
  const suggestions = useMemo(() => {
    if (!q.trim()) return [];
    const s = q.trim().toLowerCase();
    return SYMBOLS.filter(
      (x) => x.symbol.toLowerCase().includes(s) || x.name.toLowerCase().includes(s)
    ).slice(0, 8);
  }, [q]);

  /** 資產狀態 */
  const [cash, setCash] = useState(100000); // USDT
  const [positions, setPositions] = useState({}); // { BTCUSDT: { qty, cost } }
  const [orders, setOrders] = useState([]); // 開倉委託（示意）
  const [fills, setFills] = useState([]); // 成交

  /** 當前產品變更 => 重新生成假資料 */
  useEffect(() => {
    const base = clamp(lastPrice || 105000, 100, 10_000_000);
    const seed = base * (0.98 + Math.random() * 0.04);
    setKData(genCandles(seed));
    setOrderBook(makeOrderBook(seed));
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [symbol]);

  /** 假裝每 5 秒刷新五檔 */
  useEffect(() => {
    const t = setInterval(() => {
      setOrderBook(makeOrderBook((kData[kData.length - 1]?.close ?? lastPrice)));
    }, 5000);
    return () => clearInterval(t);
  }, [kData, lastPrice]);

  /** 下單（此示範直接當場成交） */
  function submitOrder({ side, price, qty }) {
    price = Number(price);
    qty = Number(qty);
    if (!(price > 0 && qty > 0)) return alert("請輸入有效價格與數量。");

    if (side === "BUY") {
      const cost = price * qty;
      if (cash < cost) return alert("餘額不足。");
      setCash((c) => c - cost);
      setPositions((p) => {
        const cur = p[symbol] || { qty: 0, cost: 0 };
        return {
          ...p,
          [symbol]: { qty: cur.qty + qty, cost: cur.cost + cost },
        };
      });
    } else {
      const pos = positions[symbol]?.qty || 0;
      if (pos < qty) return alert("持倉不足。");
      const income = price * qty;
      setCash((c) => c + income);
      setPositions((p) => {
        const cur = p[symbol];
        const remainQty = cur.qty - qty;
        const remainCost = remainQty <= 0 ? 0 : cur.cost * (remainQty / cur.qty);
        return {
          ...p,
          [symbol]: { qty: Math.max(0, remainQty), cost: remainCost },
        };
      });
    }
    const id = `${Date.now()}`;
    setFills((f) => [
      {
        id,
        ts: new Date().toISOString(),
        symbol,
        side,
        price,
        qty,
        amount: price * qty,
      },
      ...f,
    ]);
    setShowTradeForm(false);
  }

  /** 總資產（現金 + 市值） */
  const totalEquity = useMemo(() => {
    let equity = cash;
    for (const [sym, pos] of Object.entries(positions)) {
      const mark =
        sym === symbol ? lastPrice : (kData[kData.length - 1]?.close ?? lastPrice);
      equity += (pos.qty || 0) * (mark || 0);
    }
    return equity;
  }, [cash, positions, kData, lastPrice, symbol]);

  /** ====== UI 區塊 ====== */
  if (!logged) return <LoginPage onLogin={() => setLogged(true)} />;

  return (
    <div className="app">
      <header className="header">
        <button className="icon-btn" onClick={() => setDrawerOpen((v) => !v)}>
          ☰
        </button>
        <div className="brand" onClick={() => setRoute("home")}>Too Difficult</div>
        <div className="header-right">
          <div className="cash">USDT：{fmt.format(cash)}</div>
        </div>
      </header>

      <aside className={`drawer ${drawerOpen ? "open" : ""}`}>
        <nav>
          <button
            className={`drawer-item ${route === "home" ? "active" : ""}`}
            onClick={() => {
              setRoute("home");
              setDrawerOpen(false);
            }}
          >
            1. 交易頁面（首頁）
          </button>
          <button
            className={`drawer-item ${route === "assets" ? "active" : ""}`}
            onClick={() => {
              setRoute("assets");
              setDrawerOpen(false);
            }}
          >
            2. 資產總覽
          </button>
        </nav>
      </aside>

      <main className="content" onClick={() => drawerOpen && setDrawerOpen(false)}>
        {route === "home" ? (
          <Home
            q={q}
            setQ={setQ}
            suggestions={suggestions}
            onPick={(s) => {
              setSymbol(s);
              setQ("");
            }}
            symbol={symbol}
            kData={kData}
            lastPrice={lastPrice}
            orderBook={orderBook}
            showTradeForm={showTradeForm}
            setShowTradeForm={setShowTradeForm}
            onSubmitOrder={submitOrder}
          />
        ) : (
          <Assets
            totalEquity={totalEquity}
            cash={cash}
            positions={positions}
            lastPrice={lastPrice}
            symbol={symbol}
            orders={orders}
            fills={fills}
          />
        )}
      </main>
    </div>
  );
}

/** ====== 登入頁 ====== */
function LoginPage({ onLogin }) {
  const [u, setU] = useState("");
  const [p, setP] = useState("");
  return (
    <div className="login-wrap">
      <div className="login-card">
        <h1>模擬虛擬貨幣交易</h1>
        <p className="sub">請先登入以進入系統</p>
        <label>帳號</label>
        <input value={u} onChange={(e) => setU(e.target.value)} placeholder="輸入帳號" />
        <label>密碼</label>
        <input
          type="password"
          value={p}
          onChange={(e) => setP(e.target.value)}
          placeholder="輸入密碼"
        />
        <button
          className="primary"
          onClick={() => {
            if (!u || !p) return alert("請輸入帳號密碼。");
            onLogin();
          }}
        >
          登入
        </button>
      </div>
    </div>
  );
}

/** ====== 首頁 / 交易頁 ====== */
function Home({
  q,
  setQ,
  suggestions,
  onPick,
  symbol,
  kData,
  lastPrice,
  orderBook,
  showTradeForm,
  setShowTradeForm,
  onSubmitOrder,
}) {
  return (
    <>
      {/* 中央搜尋框 */}
      <div className="search-wrap">
        <input
          className="search"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="搜尋加密貨幣（例：BTC、ETH、SOL…）"
        />
        {!!suggestions.length && (
          <div className="suggest">
            {suggestions.map((s) => (
              <div
                key={s.symbol}
                className="suggest-item"
                onClick={() => onPick(s.symbol)}
              >
                <span className="sym">{s.symbol}</span>
                <span className="nm">{s.name}</span>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 交易區塊：左圖右資訊/下單 */}
      <div className="trade-grid">
        <section className="chart-wrap">
          <div className="chart-title">
            {symbol}　最新 {fmt.format(lastPrice)} USDT
          </div>
          <CandlestickChart data={kData} />
        </section>

        <section className="side-wrap">
          {!showTradeForm ? (
            <RightInfoPanel
              symbol={symbol}
              lastPrice={lastPrice}
              orderBook={orderBook}
              onTrade={() => setShowTradeForm(true)}
            />
          ) : (
            <TradeForm
              symbol={symbol}
              lastPrice={lastPrice}
              onCancel={() => setShowTradeForm(false)}
              onSubmit={onSubmitOrder}
            />
          )}
        </section>
      </div>
    </>
  );
}

function RightInfoPanel({ symbol, lastPrice, orderBook, onTrade }) {
  const chg = (Math.random() - 0.5) * 2.5; // 漲跌幅輸入
  return (
    <div className="panel">
      <div className="panel-head">
        <div>
          <div className="panel-title">{symbol}</div>
          <div className="panel-sub">現價 <span className="price-value">{fmt.format(lastPrice)}</span> USDT</div>
        </div>
        <button className="primary" onClick={onTrade}>
          交易
        </button>
      </div>

      <div className="kv">
        <div><span>24h 漲跌</span><b className={chg >= 0 ? "up" : "down"}>{chg >= 0 ? "+" : ""}{chg.toFixed(3)}%</b></div>
        <div><span>24h 最高</span><b>{fmt.format(lastPrice * 1.02)}</b></div>
        <div><span>24h 最低</span><b>{fmt.format(lastPrice * 0.98)}</b></div>
        <div><span>成交量(概估)</span><b>{fmt4.format(3200 + Math.random() * 700)} BTC</b></div>
      </div>

      <div className="orderbook">
        <div className="ob-title">五檔價格</div>
        <div className="ob-cols">
          <div>
            <div className="ob-hint">賣盤(ASK)</div>
            {orderBook.asks.slice(0).reverse().map((r, i) => (
              <div className="ob-row ask" key={`a-${i}`}>
                <span className="price">{fmt.format(r.price)}</span>
                <span className="qty">{fmt4.format(r.qty)}</span>
              </div>
            ))}
          </div>
          <div>
            <div className="ob-hint">買盤(BID)</div>
            {orderBook.bids.map((r, i) => (
              <div className="ob-row bid" key={`b-${i}`}>
                <span className="price">{fmt.format(r.price)}</span>
                <span className="qty">{fmt4.format(r.qty)}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

function TradeForm({ symbol, lastPrice, onCancel, onSubmit }) {
  const [side, setSide] = useState("BUY");
  const [price, setPrice] = useState(lastPrice.toFixed(2));
  const [qty, setQty] = useState("");

  const total = useMemo(() => {
    const p = Number(price) || 0;
    const q = Number(qty) || 0;
    return p * q;
  }, [price, qty]);

  return (
    <div className="panel">
      <div className="panel-head">
        <div>
          <div className="panel-title">{symbol} 下單</div>
          <div className="panel-sub">現價 {fmt.format(lastPrice)} USDT</div>
        </div>
        <button className="ghost" onClick={onCancel}>返回</button>
      </div>

      <div className="side-switch">
        <button
          className={`tab ${side === "BUY" ? "act" : ""}`}
          onClick={() => setSide("BUY")}
        >
          買入
        </button>
        <button
          className={`tab ${side === "SELL" ? "act" : ""}`}
          onClick={() => setSide("SELL")}
        >
          賣出
        </button>
      </div>

      <label>價格 (USDT)</label>
      <input
        value={price}
        onChange={(e) => setPrice(e.target.value)}
        inputMode="decimal"
      />

      <label>數量 ({symbol.replace("USDT", "")})</label>
      <input
        value={qty}
        onChange={(e) => setQty(e.target.value)}
        inputMode="decimal"
        placeholder="例如 0.01"
      />

      <div className="total">
        預估金額：<b>{fmt.format(total)}</b> USDT
      </div>

      <button
        className={`primary ${side === "SELL" ? "warn" : ""}`}
        onClick={() => onSubmit({ side, price, qty })}
      >
        委託{side === "BUY" ? "買進" : "賣單"}
      </button>
      <button className="ghost" onClick={onCancel}>返回</button>
    </div>
  );
}

/** ====== 資產總覽 ====== */
function Assets({ totalEquity, cash, positions, lastPrice, symbol, orders, fills }) {
  return (
    <div className="assets">
      <div className="asset-top">
        <div className="asset-title">資產總額</div>
        <div className="asset-value">{fmt.format(totalEquity)} USDT</div>
        <div className="asset-sub">可用餘額：{fmt.format(cash)} USDT</div>
      </div>

      <section className="asset-section">
        <h3>庫存（加密貨幣）</h3>
        <table className="tbl">
          <thead>
            <tr>
              <th>幣別</th>
              <th>數量</th>
              <th>持倉成本</th>
              <th>市價</th>
              <th>市值</th>
              <th>未實現損益</th>
            </tr>
          </thead>
          <tbody>
            {Object.keys(positions).length === 0 && (
              <tr><td colSpan="6" className="muted">尚無持倉</td></tr>
            )}
            {Object.entries(positions).map(([sym, pos]) => {
              const mkt = sym === symbol ? lastPrice : lastPrice; // 示意
              const mv = (pos.qty || 0) * (mkt || 0);
              const pnl = mv - (pos.cost || 0);
              return (
                <tr key={sym}>
                  <td>{sym}</td>
                  <td>{fmt4.format(pos.qty || 0)}</td>
                  <td>{fmt.format(pos.cost || 0)}</td>
                  <td>{fmt.format(mkt)}</td>
                  <td>{fmt.format(mv)}</td>
                  <td className={pnl >= 0 ? "up" : "down"}>{fmt.format(pnl)}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </section>

      <section className="asset-section">
        <h3>委託（示意）</h3>
        <table className="tbl">
          <thead>
            <tr>
              <th>時間</th><th>商品</th><th>方向</th><th>價格</th><th>數量</th><th>狀態</th>
            </tr>
          </thead>
          <tbody>
            {orders.length === 0 && (
              <tr><td colSpan="6" className="muted">無資料</td></tr>
            )}
            {orders.map((o) => (
              <tr key={o.id}>
                <td>{o.ts}</td>
                <td>{o.symbol}</td>
                <td>{o.side}</td>
                <td>{fmt.format(o.price)}</td>
                <td>{fmt4.format(o.qty)}</td>
                <td>{o.status}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="asset-section">
        <h3>成交紀錄</h3>
        <table className="tbl">
          <thead>
            <tr>
              <th>時間</th><th>商品</th><th>方向</th><th>價格</th><th>數量</th><th>金額</th>
            </tr>
          </thead>
          <tbody>
            {fills.length === 0 && (
              <tr><td colSpan="6" className="muted">尚無成交</td></tr>
            )}
            {fills.map((f) => (
              <tr key={f.id}>
                <td>{f.ts}</td>
                <td>{f.symbol}</td>
                <td>{f.side}</td>
                <td>{fmt.format(f.price)}</td>
                <td>{fmt4.format(f.qty)}</td>
                <td>{fmt.format(f.amount)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
