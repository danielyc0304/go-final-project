import React, { useEffect } from "react";
import { ArrowRight, LogIn, LineChart } from "lucide-react";
import "./Welcome.css"; // 引入剛剛建立的 CSS 檔案
import Register from "./Register.jsx";
import Login from "./Login.jsx";
import { useState } from "react";

const Welcome = ( {setLogged} ) => {
    const prevPath = localStorage.getItem("page");
    const [page, setPage] = useState("welcome");

    function changePage(newPage){
        setPage(newPage);
        localStorage.setItem("page", newPage);
    }

    function handleRegist(){
        changePage("register");
    }

    function handleLogin(){
        changePage("login");
    }

    useEffect(() => {
        console.log(`目前頁面已切換為: ${page}`);    
      }, [page]); // 依賴項

    useEffect(() => {
        if(prevPath != ""){
            setPage(prevPath);
        }
    }, []); // 依賴項


  return (
    <div>
    { (page === "welcome") ?
        <div className="welcome-container">
        {/* 背景裝飾光暈 */}
        <div className="decorative-glow glow-top"></div>
        <div className="decorative-glow glow-bottom"></div>

        {/* 主要卡片區塊 */}
        <div className="welcome-card">
            
            {/* LOGO 或 圖示區域 */}
            <div className="icon-wrapper">
            <LineChart 
                size={48} 
                color="#06b6d4" 
                className="icon-glow" 
            />
            </div>

            {/* 文字內容區 */}
            <h1 className="welcome-title">Quantis 煉金道場</h1>
            <p className="welcome-subtitle">
            在黑暗的數據海洋中，為您點亮決策之光。<br />
            專業級量化分析平台，洞察未來趨勢。
            </p>

            {/* 按鈕操作區 */}
            <div className="button-group">
            <button className="btn btn-primary" onClick={()=>handleRegist()}>
                立即註冊 <ArrowRight size={20} style={{ marginLeft: "8px" }} />
            </button>
            
            <button className="btn btn-secondary"  onClick={()=>handleLogin()}>
                <LogIn size={20} style={{ marginRight: "8px" }} />
                會員登入
            </button>
            </div>

        </div>
        
        {/* 頁腳版權宣告 */}
        <p className="welcome-footer">© 2025 Quantis Systems. All rights reserved.</p>
        </div>
        :
        (page === "login") ?
        <Login changePage={changePage} setLogged={setLogged}/>
        :
        <Register changePage={changePage}/>
    }

    </div>
  );
};

export default Welcome;