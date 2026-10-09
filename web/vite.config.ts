// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

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
    // Fonts are served as files: the server's Content-Security-Policy does
    // not allow fonts from data: URLs.
    assetsInlineLimit: (file: string) => (/\.(woff2?|ttf|otf)$/.test(file) ? false : undefined),
  },
  server: {
    port: 5173,
    proxy: { '/api': { target: 'http://127.0.0.1:18443', changeOrigin: false } },
  },
})
