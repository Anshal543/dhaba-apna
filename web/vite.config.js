import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// In dev, Vite plays the role Nginx plays in production (see nginx/default.conf.template).
// Override the targets with MENU_API_URL / ORDERS_API_URL / STATS_API_URL if needed.
const menu = process.env.MENU_API_URL || 'http://localhost:8000'
const orders = process.env.ORDERS_API_URL || 'http://localhost:8080'
const stats = process.env.STATS_API_URL || 'http://localhost:8001'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      // menu-api already serves /api/menu (and /api/menu/{id}).
      '/api/menu': { target: menu },
      '/api/orders': { target: orders, rewrite: (p) => p.replace(/^\/api\/orders/, '/orders') },
      '/api/info': { target: orders, rewrite: () => '/info' },
      '/api/stats': { target: stats },
    },
  },
})
