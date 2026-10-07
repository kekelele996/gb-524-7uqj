import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 18524,
    proxy: {
      '/api': 'http://127.0.0.1:19524',
      '/healthz': 'http://127.0.0.1:19524',
      '/readyz': 'http://127.0.0.1:19524'
    }
  }
})

