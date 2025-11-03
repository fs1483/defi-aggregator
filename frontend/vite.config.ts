import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  
  // 路径别名
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  
  // 开发服务器配置
  server: {
    port: 5175,
    host: true, // 允许外部访问
    proxy: {
      // 代理API请求到网关（开发环境）
      '/api': {
        target: 'http://localhost:5176',
        changeOrigin: true,
        secure: false,
      },
    },
  },
  
  // 构建配置
  build: {
    outDir: 'dist',
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor-react': ['react', 'react-dom'],
          'vendor-web3': ['wagmi', 'viem', 'connectkit'],
          'vendor-ui': ['lucide-react'],
        },
      },
    },
  },
  
  // 环境变量
  envPrefix: 'VITE_',
  
  // 优化配置
  optimizeDeps: {
    include: ['react', 'react-dom', 'wagmi', 'viem'],
  },
})
