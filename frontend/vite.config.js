import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    allowedHosts: ['quantis.zzppss.org'],
    host: true, 
    port: 5173,
    // API 代理設定
    proxy: {
      '/v1': {
        target: 'http://backend:8080', 
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://backend:8080',
        ws: true,
        changeOrigin: true,
      }
    },
    // Docker 
    watch: {
      usePolling: true
    }
  }
})