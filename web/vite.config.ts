import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// The build is embedded into the Go binary (internal/webui/dist).
export default defineConfig({
  plugins: [react()],
  build: {
    outDir: '../internal/webui/dist',
    emptyOutDir: true,
    sourcemap: false,
    chunkSizeWarningLimit: 900,
  },
  server: {
    port: 5173,
    proxy: { '/api': { target: 'http://127.0.0.1:18443', changeOrigin: false } },
  },
})
