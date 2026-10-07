import { defineConfig } from 'vite'
import { fileURLToPath, URL } from 'node:url'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) } },
  server: {
    host: '0.0.0.0',
    port: 8386,
    proxy: { '/health': process.env.VITE_API_PROXY_TARGET ?? 'http://localhost:6868', '/ready': process.env.VITE_API_PROXY_TARGET ?? 'http://localhost:6868', '/api': process.env.VITE_API_PROXY_TARGET ?? 'http://localhost:6868' },
  },
})
