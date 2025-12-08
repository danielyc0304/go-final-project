import React, { useState } from "react";
import { User, Mail, Lock, ArrowRight } from "lucide-react";
import "./Register.css";

const Register = ({ changePage }) => { 
  
  // 1. 定義表單狀態
  const [formData, setFormData] = useState({
    email: "",
    name: "",
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
    console.log("註冊資料:", formData);

    try {
      console.log("Register working...");
      console.log("Register working...");
      
      // 取得 API 基礎網址
      const API_BASE_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";
      
      const response = await fetch(`${API_BASE_URL}/v1/auth/registration`, {
        method: "POST",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
        },
        // body: JSON.stringify({ username: idNumber, password: passwd }),
        body: JSON.stringify(formData),
      });

      if (!response.ok) {
        if (response.status === 409) {
          console.error('帳號已經存在');
          // 在這裡處理 409 錯誤的邏輯，例如顯示特定的用戶訊息
          alert('該項目帳號已經註冊');
          changePage("login");
          return;
        }
        else {
            throw new Error("Network response was not ok");
        }
      }

      const data = await response.json();

      console.log(data)

      if (data.success) {
        console.log("註冊成功");
        alert("註冊成功");
        changePage("login")
      } else {
        console.error("註冊失敗");
        alert("註冊失敗，請稍後再試");
      }
    } catch (error) {
      console.error("Registration failed with error");
      console.log(error);

      alert("Registration failed with error");
    }

  };

  return (
    <div className="register-container">
      {/* 背景裝飾 */}
      <div className="decorative-glow glow-top-left"></div>
      <div className="decorative-glow glow-bottom-right"></div>

      <div className="register-card">
        <header className="register-header">
          <h1 className="register-title">加入 Quantis</h1>
          <p className="register-subtitle">建立您的專業量化分析帳號</p>
        </header>

        <form className="register-form" onSubmit={handleSubmit}>
          
          {/* 姓名欄位 */}
          <div className="input-group">
            <input
              type="text"
              name="name"
              placeholder="您的稱呼 (Name)"
              className="input-field"
              value={formData.name}
              onChange={handleChange}
              required
            />
            <User size={20} className="input-icon" />
          </div>

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
              placeholder="設定密碼 (Password)"
              className="input-field"
              value={formData.password}
              onChange={handleChange}
              required
            />
            <Lock size={20} className="input-icon" />
          </div>

          {/* 註冊按鈕 */}
          <button type="submit" className="btn-submit">
            立即註冊 <ArrowRight size={18} style={{ display: 'inline', verticalAlign: 'text-bottom' }} />
          </button>
        </form>

        <div className="register-footer">
          已經有帳號了嗎？ 
          {/* 如果有傳入切換函數則使用，否則只顯示樣式 */}
          <span 
            className="link-login" 
            // onClick={()=>setPage("login")}
            onClick={() => changePage("login")}
          >
            登入
          </span>
        </div>
      </div>
    </div>
  );
};

export default Register;