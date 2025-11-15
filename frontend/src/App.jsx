import React, { useEffect, useMemo, useRef, useState } from "react";
import "./App.css";

/* ===== 小工具 ===== */
const fmt = new Intl.NumberFormat("zh-TW", { maximumFractionDigits: 2 });
const fmt4 = new Intl.NumberFormat("zh-TW", { maximumFractionDigits: 6 });
const nowSec = () => Math.floor(Date.now() / 1000);

/* 產生初始 K 線：1 分鐘一根 */
function genCandles(seedPrice = 105000, n = 240) {
  const out = [];
  let p = seedPrice;
  const t0 = nowSec() - n * 60;
  for (let i = 0; i < n; i++) {
    const time = t0 + i * 60;
    const drift = (Math.random() - 0.5) * (p * 0.0016);
    const open = p;
    const close = open + drift; 
    const high = Math.max(open, close) + Math.random() * (p * 0.0008);
    const low = Math.min(open, close) - Math.random() * (p * 0.0008);
    out.push({
      time,
      open: Number(open.toFixed(2)),
      high: Number(high.toFixed(2)),
      low: Number(low.toFixed(2)),
      close: Number(close.toFixed(2)),
    });
    p = close;
  }
  return out;
}

/* 以現價為中心產生 N 檔：BID 高→低、ASK 低→高 */
function makeOrderBookAround(mid, levels = 6) {
  const step = Math.max(1, Math.round(mid * 0.0008));
  const bestBid = Math.floor(mid / step) * step;
  const bestAsk = bestBid + step;

  const bids = Array.from({ length: levels }, (_, i) => ({
    price: bestBid - i * step,
    qty: Number((Math.random() * 0.9 + 0.1).toFixed(4)),
  }));

  const asks = Array.from({ length: levels }, (_, i) => ({
    price: bestAsk + i * step,
    qty: Number((Math.random() * 0.9 + 0.1).toFixed(4)),
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

/* ===== K 線元件 ===== */
function CandlestickChart({ data }) {
  const containerRef = useRef(null);
  const chartRef = useRef(null);
  const seriesRef = useRef(null);

  // 第一次掛載：建立 chart
  useEffect(() => {
    let chart;
    let series;
    let ro;

    (async () => {
      const { createChart } = await import("lightweight-charts");
      const el = containerRef.current;
      if (!el) return;

      chart = createChart(el, {
        width: el.clientWidth,
        height: 560,
        layout: {
          background: { type: "solid", color: "#0f1115" },
          textColor: "#e6e6e6",
        },
        grid: {
          vertLines: { color: "#1b1f2a" },
          horzLines: { color: "#1b1f2a" },
        },
        rightPriceScale: { borderColor: "#2a2f3b" },
        timeScale: {
          borderColor: "#2a2f3b",
          timeVisible: true,
          secondsVisible: false,
        },
        crosshair: { mode: 1 },
      });

      series = chart.addCandlestickSeries({
        upColor: "#26a69a",
        downColor: "#ef5350",
        wickUpColor: "#26a69a",
        wickDownColor: "#ef5350",
        borderVisible: false,
      });

      series.setData(data);

      chartRef.current = chart;
      seriesRef.current = series;

      // 自適應寬度
      ro = new ResizeObserver(() => {
        if (!containerRef.current) return;
        chart.applyOptions({ width: containerRef.current.clientWidth });
      });
      ro.observe(el);
    })();

    return () => {
      if (ro) ro.disconnect?.();
      chart?.remove?.();
      chartRef.current = null;
      seriesRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // 只在第一次掛載時建立 chart

  // 每次資料更新時更新 K 線
  useEffect(() => {
    if (!seriesRef.current) return;
    seriesRef.current.setData(data);
  }, [data]);

  return <div className="chart" ref={containerRef} />;
}


/* ===== 主應用 ===== */
export default function App() {
  const [logged, setLogged] = useState(false);

  const [symbol, setSymbol] = useState("BTCUSDT");
  const [kData, setKData] = useState(() => genCandles(105000));
  const [orderBook, setOrderBook] = useState(() =>
    makeOrderBookAround(105000, 6)
  );
  const lastPrice = kData[kData.length - 1]?.close ?? 0;

  // 現價漲跌顏色
  const prevRef = useRef(lastPrice);
  const priceTrend =
    lastPrice > prevRef.current ? "up" : lastPrice < prevRef.current ? "down" : "";
  useEffect(() => {
    prevRef.current = lastPrice;
  }, [lastPrice]);

  // 搜尋
  const [q, setQ] = useState("");
  const suggestions = useMemo(() => {
    if (!q.trim()) return [];
    const s = q.trim().toLowerCase();
    return SYMBOLS.filter(
      (x) => x.symbol.toLowerCase().includes(s) || x.name.toLowerCase().includes(s)
    ).slice(0, 8);
  }, [q]);

  // 資金 / 倉位
  const [cash, setCash] = useState(100000); // 可用資金 + 已實現損益
  const [positions, setPositions] = useState([]); // {id,symbol,side,qty,entry,leverage,notional,margin,closePrice,tp?,sl?,orderType}
  const [fills, setFills] = useState([]); // 目前沒顯示，但保留
  const [realized, setRealized] = useState(0); // 已實現損益

  // 換商品 → 新 K 線
  useEffect(() => {
    const seed = (lastPrice || 105000) * (0.985 + Math.random() * 0.03);
    setKData(genCandles(seed));
  }, [symbol]);

  // 讓 K 線/現價每秒跳動
  useEffect(() => {
    const t = setInterval(() => {
      setKData((arr) => {
        if (!arr.length) return arr;
        const out = [...arr];
        const last = out[out.length - 1];
        const now = nowSec();

        const step = Math.max(0.01, last.close * 0.0006 * (1 + Math.random() * 0.5));
        const change = (Math.random() * 2 - 1) * step;
        const nextClose = Number((last.close + change).toFixed(2));

        if (now < last.time + 60) {
          const high = Number(Math.max(last.high, nextClose).toFixed(2));
          const low = Number(Math.min(last.low, nextClose).toFixed(2));
          out[out.length - 1] = { ...last, close: nextClose, high, low };
        } else {
          const open = last.close;
          const close = nextClose;
          const high = Number(Math.max(open, close).toFixed(2));
          const low = Number(Math.min(open, close).toFixed(2));
          out.push({
            time: last.time + 60,
            open: Number(open.toFixed(2)),
            high,
            low,
            close,
          });
          if (out.length > 240) out.shift();
        }
        return out;
      });
    }, 1000);
    return () => clearInterval(t);
  }, []);

  // 五檔綁定現價
  useEffect(() => {
    if (lastPrice > 0) setOrderBook(makeOrderBookAround(lastPrice, 6));
  }, [lastPrice]);

  // 下單：由 TradePanel 算好 qtyCoin / notional / margin
  function submitOrder({
    side,
    price,
    orderType,
    qtyCoin,
    leverage,
    tpOn,
    tp,
    slOn,
    sl,
    notional,
    margin,
  }) {
    const entry = Number(price);
    const lev = Number(leverage);
    const quantityCoin = Number(qtyCoin);
    if (!(entry > 0 && lev >= 1 && quantityCoin > 0 && notional > 0 && margin > 0)) {
      return alert("請輸入有效價格、槓桿與數量。");
    }
  
    // 檢查可用餘額是否足夠支付保證金
    if (cash < margin) {
      alert("可用餘額不足，無法開倉。");
      return;
    }
  
    // 扣掉保證金
    setCash((c) => c - margin);
  
    const closePrice =
      side === "BUY" ? entry * (1 - 1 / lev) : entry * (1 + 1 / lev); // 示意平倉價

    const id = `${Date.now()}`;
    setPositions((ps) => [
      {
        id,
        symbol,
        side,
        qty: quantityCoin,
        entry,
        leverage: lev,
        notional, // 槓桿後倉位大小
        margin, // 未槓桿前投入金額（保證金）
        closePrice,
        tp: tpOn ? Number(tp) || null : null,
        sl: slOn ? Number(sl) || null : null,
        orderType,
      },
      ...ps,
    ]);
    setFills((f) => [
      {
        id,
        ts: new Date().toISOString(),
        symbol,
        side,
        price: entry,
        qty: quantityCoin,
        amount: notional,
      },
      ...f,
    ]);
  }

  const markPrice = lastPrice;
  // 未實現損益
  const unreal = useMemo(
    () =>
      positions.reduce(
        (u, p) =>
          u +
          (markPrice - p.entry) * p.qty * (p.side === "BUY" ? 1 : -1),
        0
      ),
    [positions, markPrice]
  );

  // 倉位總額：所有倉位的名目（notional）加總
  const totalPositionValue = useMemo(
    () => positions.reduce((sum, p) => sum + (p.qty || 0) * markPrice, 0),
    [positions, markPrice]
  );


  function closePosition(pid) {
    const p = positions.find((x) => x.id === pid);
    if (!p) return;
    const pnl =
      (markPrice - p.entry) * p.qty * (p.side === "BUY" ? 1 : -1);
  
    // 把保證金 + 損益一起加回可用餘額
    setCash((c) => c + p.margin + pnl);
    setRealized((r) => r + pnl);
    setPositions((ps) => ps.filter((x) => x.id !== pid));
  }

  if (!logged) return <LoginPage onLogin={() => setLogged(true)} />;

  return (
    <div className="app">
      <header className="header">
        <div className="brand">Quantis</div>
        <div className="header-right">USDT：{fmt.format(cash)}</div>
      </header>

      <main className="content">
        {/* 搜尋 */}
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
                  onClick={() => {
                    setSymbol(s.symbol);
                    setQ("");
                  }}
                >
                  <span className="sym">{s.symbol}</span>
                  <span className="nm">{s.name}</span>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* 左：K 線、中：報價、右：下單 */}
        <div className="trade-3col">
          <section className="chart-wrap">
            <div className="chart-title">{symbol}</div>
            <CandlestickChart data={kData} />
          </section>

          <OrderBookPanel
            symbol={symbol}
            lastPrice={lastPrice}
            trend={priceTrend}
            orderBook={orderBook}
          />

          <TradePanel
            symbol={symbol}
            lastPrice={lastPrice}
            onSubmit={submitOrder}
          />
        </div>

        {/* 下方倉位總覽 */}
        <PositionsTable
          positions={positions}
          markPrice={markPrice}
          closePosition={closePosition}
          totalPositionValue={totalPositionValue}
          unreal={unreal}
          realized={realized}
        />
      </main>
    </div>
  );
}

/* ===== 登入頁 ===== */
function LoginPage({ onLogin }) {
  const [u, setU] = useState("");
  const [p, setP] = useState("");
  return (
    <div className="login-wrap">
      <div className="login-card">
        <h1>模擬虛擬貨幣交易</h1>
        <p className="sub">請先登入以進入系統</p>
        <label>帳號</label>
        <input
          value={u}
          onChange={(e) => setU(e.target.value)}
          placeholder="輸入帳號"
        />
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

/* ===== 報價 / 五檔 ===== */
function OrderBookPanel({ symbol, lastPrice, trend, orderBook }) {
  const chg = (Math.random() - 0.5) * 2.3;
  return (
    <section className="quote-wrap">
      <div className="panel">
        <div className="panel-head">
          <div className="panel-title">{symbol}</div>
        </div>

        <div className="quote-price">
          現價{" "}
          <span
            className={`price-value ${
              trend === "up" ? "price-up" : trend === "down" ? "price-down" : ""
            }`}
          >
            {fmt.format(lastPrice)}
          </span>{" "}
          USDT
        </div>

        <div className="kv">
          <div>
            <span>24h 漲跌</span>
            <b className={chg >= 0 ? "up" : "down"}>
              {chg >= 0 ? "+" : ""}
              {chg.toFixed(2)}%
            </b>
          </div>
          <div>
            <span>24h 最高</span>
            <b>{fmt.format(lastPrice * 1.02)}</b>
          </div>
          <div>
            <span>24h 最低</span>
            <b>{fmt.format(lastPrice * 0.98)}</b>
          </div>
          <div>
            <span>成交量(估)</span>
            <b>
              {fmt4.format(3000 + Math.random() * 800)}{" "}
              {symbol.replace("USDT", "")}
            </b>
          </div>
        </div>

        <div className="orderbook">
          <div className="ob-cols">
            <div>
              <div className="ob-hint">買進(BID)</div>
              {orderBook.bids.map((r, i) => (
                <div className="ob-row bid" key={`b-${i}`}>
                  <span className="price">{fmt.format(r.price)}</span>
                  <span className="qty">{fmt4.format(r.qty)}</span>
                </div>
              ))}
            </div>
            <div>
              <div className="ob-hint">賣出(ASK)</div>
              {orderBook.asks.map((r, i) => (
                <div className="ob-row ask" key={`a-${i}`}>
                  <span className="price">{fmt.format(r.price)}</span>
                  <span className="qty">{fmt4.format(r.qty)}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

/* ===== 下單面板（限價 / 市價 + 槓桿 / 名目 / 保證金 / TP/SL） ===== */
function TradePanel({ symbol, lastPrice, onSubmit }) {
  const [side, setSide] = useState("BUY");
  const [orderType, setOrderType] = useState("LIMIT"); // LIMIT | MARKET
  const [price, setPrice] = useState(lastPrice.toFixed(2));
  const [inputMode, setInputMode] = useState("COIN"); // COIN | USD
  const [qty, setQty] = useState("");
  const [lev, setLev] = useState(10);
  const [tpOn, setTpOn] = useState(false);
  const [slOn, setSlOn] = useState(false);
  const [tp, setTp] = useState("");
  const [sl, setSl] = useState("");

  // 限價時才自動跟隨最新價
  useEffect(() => {
    if (orderType === "LIMIT" && lastPrice > 0) {
      setPrice(lastPrice.toFixed(2));
    }
  }, [orderType]);

  const parsedPrice = Number(price) || 0;
  const parsedQty = Number(qty) || 0;

  // 市價單使用現價，限價單使用輸入價格
  const basePrice = orderType === "MARKET" ? lastPrice : parsedPrice;

  let margin = 0;
  let notional = 0;
  let coinQty = 0;

  if (basePrice > 0 && parsedQty > 0 && lev > 0) {
    if (inputMode === "USD") {
      // USD 模式：輸入為保證金
      margin = parsedQty;
      notional = margin * lev; // 槓桿後倉位金額
      coinQty = notional / basePrice; // 換算幣數
    } else {
      // 幣數模式：輸入為幣數
      coinQty = parsedQty;
      notional = coinQty * basePrice; // 槓桿前名目
      margin = notional / lev; // 槓桿所需保證金
    }
  }

  const closePrice =
    basePrice > 0 && lev > 0
      ? side === "BUY"
        ? basePrice * (1 - 1 / lev)
        : basePrice * (1 + 1 / lev)
      : 0;

  return (
    <section className="trade-wrap">
      <div className="panel">
        <div className="panel-head">
          <div className="panel-title">下單</div>
        </div>

        {/* 買進 / 賣出 */}
        <div className="side-switch">
          <button
            className={`tab ${side === "BUY" ? "act" : ""}`}
            onClick={() => setSide("BUY")}
          >
            買進
          </button>
          <button
            className={`tab ${side === "SELL" ? "act" : ""}`}
            onClick={() => setSide("SELL")}
          >
            賣出
          </button>
        </div>

        {/* 限價 / 市價 */}
        <div className="order-type-row">
          <span className="order-type-label">下單方式</span>
          <div className="order-type-switch">
            <button
              className={`mini-tab ${orderType === "LIMIT" ? "act" : ""}`}
              onClick={() => setOrderType("LIMIT")}
            >
              限價
            </button>
            <button
              className={`mini-tab ${orderType === "MARKET" ? "act" : ""}`}
              onClick={() => setOrderType("MARKET")}
            >
              市價
            </button>
          </div>
        </div>

        <label>價格 (USDT)</label>
        <input
          value={orderType === "MARKET" ? "" : price}
          onChange={(e) => setPrice(e.target.value)}
          inputMode="decimal"
          disabled={orderType === "MARKET"}
          placeholder={
            orderType === "MARKET"
              ? `${fmt.format(lastPrice)}（市價）`
              : ""
          }
        />

        <div className="qty-row">
          <label>
            {inputMode === "COIN"
              ? `數量 (${symbol.replace("USDT", "")})`
              : "保證金 (USDT)"}
          </label>
          <button
            className="mini-switch"
            onClick={() =>
              setInputMode((m) => (m === "COIN" ? "USD" : "COIN"))
            }
          >
            切換為 {inputMode === "COIN" ? "USDT" : "幣數"}
          </button>
        </div>
        <input
          value={qty}
          onChange={(e) => setQty(e.target.value)}
          inputMode="decimal"
          placeholder={inputMode === "COIN" ? "例如 0.01" : "例如 100"}
        />

        <label>
          槓桿：{lev}x
          {closePrice ? (
            <span className="lev-hint">（估平倉價：{fmt.format(closePrice)}）</span>
          ) : null}
        </label>

        <input
          type="range"
          min={1}
          max={50}
          step={1}
          value={lev}
          onChange={(e) => setLev(Number(e.target.value))}
        />

        {/* TP / SL */}
        <div className="tpsl-row">
          <label className="inline">
            <input
              type="checkbox"
              checked={tpOn}
              onChange={(e) => setTpOn(e.target.checked)}
            />
            TP
          </label>
          <input
            disabled={!tpOn}
            value={tp}
            onChange={(e) => setTp(e.target.value)}
            placeholder="TP 觸發價"
          />
        </div>
        <div className="tpsl-row">
          <label className="inline">
            <input
              type="checkbox"
              checked={slOn}
              onChange={(e) => setSlOn(e.target.checked)}
            />
            SL
          </label>
          <input
            disabled={!slOn}
            value={sl}
            onChange={(e) => setSl(e.target.value)}
            placeholder="SL 觸發價"
          />
        </div>

        {/* 名目 / 平倉價顯示 */}
        <label>名目 (USDT)</label>
        <div className="display-box right">{fmt.format(notional)} USDT</div>

        <button
          className={`primary ${side === "SELL" ? "warn" : ""}`}
          onClick={() => {
            if (!(basePrice > 0 && coinQty > 0 && notional > 0 && margin > 0)) {
              alert("請輸入有效價格、槓桿與數量。");
              return;
            }
            onSubmit({
              side,
              price: basePrice,
              orderType,
              qtyCoin: coinQty,
              leverage: lev,
              tpOn,
              tp,
              slOn,
              sl,
              notional,
              margin,
            });
          }}
        >
          送出{side === "BUY" ? "買進" : "賣出"}
        </button>
      </div>
    </section>
  );
}

/* ===== 倉位表（含 限價 / 市價 標籤） ===== */
function PositionsTable({
  positions,
  markPrice,
  closePosition,
  totalPositionValue,
  unreal,
  realized,
}) {
  return (
    <section className="positions-wrap">
      <div className="asset-top mini">
        <div className="asset-title">倉位總額</div>
        <div className="asset-value">{fmt.format(totalPositionValue)} USDT</div>
        <div className="asset-sub">
          未實現損益：
          <span className={unreal >= 0 ? "up" : "down"}>
            {fmt.format(unreal)}
          </span>
        </div>
        <div className="asset-sub">
          已實現損益：
          <span className={realized >= 0 ? "up" : "down"}>
            {fmt.format(realized)}
          </span>
        </div>
      </div>

      <table className="tbl">
        <thead>
          <tr>
            <th>倉位</th>
            <th>數量</th>
            <th>名目價值</th>
            <th>保證金</th>
            <th>買入價</th>
            <th>市場價</th>
            <th>平倉價</th>
            <th>未實現損益</th>
            <th>TP</th>
            <th>SL</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {positions.length === 0 && (
            <tr>
              <td colSpan="11" className="muted">
                尚無倉位
              </td>
            </tr>
          )}
          {positions.map((p) => {
            const pnl =
              (markPrice - p.entry) * p.qty * (p.side === "BUY" ? 1 : -1);
            const orderTypeLabel =
              p.orderType === "MARKET" ? "市價" : "限價";
            return (
              <tr key={p.id}>
                <td>
                  {p.symbol}{" "}
                  <span className={p.side === "BUY" ? "up" : "down"}>
                    {p.side === "BUY" ? "買進" : "賣出"}
                  </span>{" "}
                  {p.leverage}x{" "}
                  <span className="order-type-tag">{orderTypeLabel}</span>
                </td>
                <td>{fmt4.format(p.qty)}</td>
                <td>{fmt.format(p.notional)}</td>
                <td>{fmt.format(p.margin)}</td>
                <td>{fmt.format(p.entry)}</td>
                <td>{fmt.format(markPrice)}</td>
                <td>{fmt.format(p.closePrice)}</td>
                <td className={pnl >= 0 ? "up" : "down"}>
                  {fmt.format(pnl)}
                </td>
                <td>{p.tp ? fmt.format(p.tp) : "-"}</td>
                <td>{p.sl ? fmt.format(p.sl) : "-"}</td>
                <td>
                  <button
                    className="ghost"
                    onClick={() => closePosition(p.id)}
                  >
                    平倉
                  </button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </section>
  );
}
