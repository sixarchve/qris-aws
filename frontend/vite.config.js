import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const backendTarget = process.env.VITE_BACKEND_TARGET || 'http://localhost:8080'
const websocketTarget = process.env.VITE_WS_TARGET || backendTarget.replace(/^http/, 'ws')

export default defineConfig({
  base: '/merchant/',
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      '/api': {
        target: backendTarget,
        changeOrigin: true,
      },
      '/ws': {
        target: websocketTarget,
        ws: true,
        changeOrigin: true,
      }
    }
  }
})
