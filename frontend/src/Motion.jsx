import React, { useEffect, useState, useRef, useMemo } from 'react';
import { Mail, Lock, ArrowRight } from "lucide-react";


const Motion = () => {
  // --- 參數設定 ---
  const MAX_CANDLES = 40;     // 畫面上保留幾根 K 棒
  const CANDLE_DURATION = 5; // 每一根 K 棒要跳動幾次才收盤 (數值越小K棒換得越快)
  const UPDATE_SPEED = 10;     // 畫面更新速度 (每N幀更新一次數據)
  const VOLATILITY = 5;       // 波動幅度
  
  // 生成一根隨機 K 棒
  const generateCandle = (startPrice) => {
    // 簡單模擬：先隨機決定收盤，再隨機決定上下影線
    const move = (Math.random() - 0.5) * VOLATILITY;
    const close = startPrice + move;
    const open = startPrice;
    const high = Math.max(open, close) + Math.random() * 2;
    const low = Math.min(open, close) - Math.random() * 2;
    return { open, close, high, low, id: Math.random() };
  };

  // 初始化歷史數據
  const initData = () => {
    let currentPrice = 100;
    const initial = [];
    for (let i = 0; i < MAX_CANDLES; i++) {
      const candle = generateCandle(currentPrice);
      initial.push(candle);
      currentPrice = candle.close;
    }
    return initial;
  };

  const [history, setHistory] = useState(initData());
  
  // "Live Candle" 狀態：這是目前最右邊正在跳動的那一根
  const [liveCandle, setLiveCandle] = useState(() => {
    const lastClose = history[history.length - 1].close;
    return { open: lastClose, close: lastClose, high: lastClose, low: lastClose };
  });

  const requestRef = useRef();
  const tickCount = useRef(0);      // 控制更新頻率
  const candleTickCount = useRef(0); // 控制這根 K 棒存活多久

  useEffect(() => {
    const animate = () => {
      tickCount.current += 1;
      
      if (tickCount.current % UPDATE_SPEED === 0) {
        setLiveCandle(prevLive => {
          // 1. 隨機跳動價格
          const move = (Math.random() - 0.48) * 3; // 稍微偏向多頭 (0.48 < 0.5)
          const newClose = prevLive.close + move;
          
          // 2. 更新最高與最低 (High/Low)
          const newHigh = Math.max(prevLive.high, newClose);
          const newLow = Math.min(prevLive.low, newClose);
          
          const updatedLive = { 
            ...prevLive, 
            close: newClose, 
            high: newHigh, 
            low: newLow 
          };

          // 3. 檢查是否該「收盤」換下一根
          candleTickCount.current += 1;
          if (candleTickCount.current > CANDLE_DURATION) {
            // 將這根完成的 K 棒推入歷史
            setHistory(prevHistory => {
              const newHistory = [...prevHistory.slice(1), updatedLive];
              return newHistory;
            });
            
            // 重置計數器，並以當前收盤價作為下一根的開盤價
            candleTickCount.current = 0;
            return { 
              open: newClose, 
              close: newClose, 
              high: newClose, 
              low: newClose 
            };
          }

          return updatedLive;
        });
      }
      requestRef.current = requestAnimationFrame(animate);
    };

    requestRef.current = requestAnimationFrame(animate);
    return () => cancelAnimationFrame(requestRef.current);
  }, []);

  // --- 計算繪圖與鏡頭 ---
  const { candlesToRender, viewBox, lastPrice } = useMemo(() => {
    // 合併歷史數據與當前正在跳動的 K 棒
    const allCandles = [...history, liveCandle];
    
    // 計算畫面最高與最低價，用於自動縮放 (Auto-Fit)
    let minLow = Infinity;
    let maxHigh = -Infinity;
    
    allCandles.forEach(c => {
      if (c.low < minLow) minLow = c.low;
      if (c.high > maxHigh) maxHigh = c.high;
    });

    // 增加邊界緩衝
    const padding = (maxHigh - minLow) * 0.2; 
    const vMin = minLow - padding;
    const vHeight = (maxHigh - minLow) + (padding * 2);

    // X 軸設定
    const CANDLE_WIDTH = 2; // K棒寬度
    const GAP = 1;          // 間距
    const totalWidth = allCandles.length * (CANDLE_WIDTH + GAP);

    return {
      candlesToRender: allCandles,
      viewBox: `0 ${vMin} ${totalWidth} ${vHeight}`,
      lastPrice: liveCandle.close.toFixed(2)
    };
  }, [history, liveCandle]);

  return (
    <div style={styles.container}>
      {/* 背景網格 */}
      <div style={styles.grid}></div>

      <svg 
        viewBox={viewBox} 
        preserveAspectRatio="none" 
        style={styles.svg}
      >
        <defs>
          {/* 螢光濾鏡 */}
          <filter id="glow" x="-50%" y="-50%" width="200%" height="200%">
            <feGaussianBlur stdDeviation="0.5" result="blur" />
            <feComposite in="SourceGraphic" in2="blur" operator="over" />
          </filter>
        </defs>

        {/* 繪製每一根 K 棒 */}
        {candlesToRender.map((c, i) => {
          const x = i * 3; // (Width 2 + Gap 1)
          const isUp = c.close >= c.open;
          const color = isUp ? '#10b981' : '#ef4444'; // 綠漲紅跌
          
          // 實體長度 (Body)
          // SVG rect height 不能為負，所以要算 top 和 height
          const bodyTop = Math.min(c.open, c.close);
          const bodyHeight = Math.max(Math.abs(c.close - c.open), 0.2); // 0.2 避免開收同價時消失

          return (
            <g key={i} filter="url(#glow)">
              {/* 上下影線 (Wick) */}
              <line 
                x1={x + 1} y1={c.low} 
                x2={x + 1} y2={c.high} 
                stroke={color} 
                strokeWidth="0.2" 
              />
              {/* 實體 (Body) */}
              <rect 
                x={x} 
                y={bodyTop} 
                width="2" 
                height={bodyHeight} 
                fill={color} 
              />
            </g>
          );
        })}
        
        {/* 最新價格線 */}
        <line 
          x1="0" 
          y1={Number(lastPrice)} 
          x2="1000" // 拉很長
          y2={Number(lastPrice)} 
          stroke="rgba(255, 255, 255, 0.3)" 
          strokeDasharray="2" 
          strokeWidth="0.2"
        />
      </svg>

      {/* <div style={styles.overlay}>
        <div style={styles.priceBox}>
          <span style={{color: '#aaa', fontSize: '12px'}}>BTC/USD</span>
          <br/>
          <span style={styles.price}>{lastPrice}</span>
        </div>
      </div> */}

    <div style={styles.cardbg}>
      <div className="login-card" >
        <header className="login-header">
          <h1 className="login-title">歡迎回來</h1>
          <p className="login-subtitle">登入以繼續使用 Quantis 服務</p>
        </header>

        <form className="login-form">
          
          {/* Email 欄位 */}
          <div className="input-group">
            <input
              type="email"
              name="email"
              placeholder="電子郵件 (Email)"
              className="input-field"
              value=""
              required
            />
            <Mail size={20} className="input-icon" />
          </div>

          {/* 密碼欄位 */}
          <div className="input-group">
            <input
              type="password"
              name="password"
              placeholder="您的密碼 (Password)"
              className="input-field"
              value=""
              required
            />
            <Lock size={20} className="input-icon" />
          </div>

          {/* 登入按鈕 */}
          <button type="submit" className="btn-submit">
            立即登入 <ArrowRight size={18} style={{ display: 'inline', verticalAlign: 'text-bottom' }} />
          </button>
        </form>

        <div className="login-footer">
          還沒有帳號嗎？ 
          <span 
            className="link-register" 
          >
            立即註冊
          </span>
        </div>
      </div>
    </div>  

    </div>
  );
};

const styles = {
  cardbg: {
      height: "100vh",
      width: "100%",
      display: "flex",
      justifyContent: "center",
      alignItems: "center",
      position: "relative",
      overflow: "hidden",
  },
  container: {
    position: 'relative',
    width: '100%',
    height: '100vh',
    backgroundColor: '#09090b', // 炭黑色背景
    overflow: 'hidden',
    fontFamily: 'Inter, system-ui, sans-serif',
  },
  svg: {
    position: 'absolute',
    bottom: 0, left: 0, width: '100%', height: '100%',
    // 鏡頭平滑移動
    transition: 'viewBox 0.3s ease-out', 
  },
  grid: {
    position: 'absolute',
    top: 0, left: 0, width: '100%', height: '100%',
    backgroundImage: `linear-gradient(#1f2937 1px, transparent 1px), linear-gradient(90deg, #1f2937 1px, transparent 1px)`,
    backgroundSize: '40px 40px',
    opacity: 0.3,
  },
  overlay: {
    position: 'absolute',
    top: 20, right: 20,
    zIndex: 10,
  },
  priceBox: {
    backgroundColor: 'rgba(255,255,255,0.1)',
    backdropFilter: 'blur(5px)',
    padding: '10px 20px',
    borderRadius: '8px',
    textAlign: 'right',
    border: '1px solid rgba(255,255,255,0.1)',
  },
  price: {
    color: '#fff',
    fontSize: '24px',
    fontWeight: 'bold',
    fontVariantNumeric: 'tabular-nums', // 讓數字等寬，跳動時不抖動
  }
};

export default Motion;