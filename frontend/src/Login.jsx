import React, { useState } from "react";
import { Mail, Lock, ArrowRight } from "lucide-react";
import "./Login.css"; // 確保引入了樣式檔

const Login = ({ changePage, setLogged }) => { 
  
  // 1. 定義表單狀態 (登入僅需 Email 與 密碼)
  const [formData, setFormData] = useState({
    email: "",
    password: ""
  });

  // 2. 處理輸入變更
  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));
  };

  // 3. 處理表單送出
  const handleSubmit = async (e) => {
    e.preventDefault();
    console.log("登入資料:", formData);

    try {
      // 修改為 Login API
      const response = await fetch("http://localhost:8080/v1/auth/login", {
        method: "POST",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(formData),
      });

      if (!response.ok) {
        if(response.status == 401){
          alert("登入失敗：" + "請檢查帳號密碼");
        }
        else {
          throw new Error("Network response was not ok");
        }
      }

      const data = await response.json();

      if (data.success) {
        alert("登入成功！");
        // 登入成功後的跳轉邏輯，例如跳轉到首頁
        // setPage("dashboard"); 
        localStorage.setItem("token", data.data.token);
        localStorage.setItem("page", "welcome");
        setLogged(true);
      } else {
        // alert("登入失敗：" + (data.message || "請檢查帳號密碼"));
      }
    } catch (error) {
      console.error("Login error:", error);
      alert("登入發生錯誤，請稍後再試");
    }
  };

  return (
    <div className="login-container">
    {/* 背景光暈效果 */}
      <div className="decorative-glow glow-top-left"></div>
      <div className="decorative-glow glow-bottom-right"></div>

      <div className="login-card">
        <header className="login-header">
          <h1 className="login-title">歡迎回來</h1>
          <p className="login-subtitle">登入以繼續使用 Quantis 服務</p>
        </header>

        <form className="login-form" onSubmit={handleSubmit}>
          
          {/* Email 欄位 */}
          <div className="input-group">
            <input
              type="email"
              name="email"
              placeholder="電子郵件 (Email)"
              className="input-field"
              value={formData.email}
              onChange={handleChange}
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
              value={formData.password}
              onChange={handleChange}
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
            onClick={() => changePage("register")}
          >
            立即註冊
          </span>
        </div>
      </div>
    </div>
  );
};

export default Login;