/* global process */
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// The browser always calls the same-origin `/api/...` path (see src/api.js).
// In `vite dev` there is no nginx in front, so proxy `/api` to the local API
// service and strip the prefix to match the Go router's root-mounted routes.
const devApiTarget = process.env.VITE_DEV_API_TARGET || 'http://localhost:8080'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': {
        target: devApiTarget,
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})
