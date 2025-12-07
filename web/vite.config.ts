import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    host: true,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: (id) => {
          // 将 Recharts 单独分割
          if (id.includes('recharts')) {
            return 'recharts';
          }
          // 将 MUI 相关库分割
          if (id.includes('@mui/material') || id.includes('@mui/icons-material') || id.includes('@mui/x-date-pickers')) {
            return 'mui';
          }
          // 将 node_modules 中的其他依赖分割
          if (id.includes('node_modules')) {
            return 'vendor';
          }
        },
      },
    },
  },
})
