import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')

  return {
    plugins: [react()],
    server: {
      port: 5173,
      proxy: {
        '/auth': env.VITE_API_BASE_URL || 'http://localhost:8080',
        '/match': env.VITE_API_BASE_URL || 'http://localhost:8080',
        '/daily-challenge': env.VITE_API_BASE_URL || 'http://localhost:8080',
        '/daily-submit': env.VITE_API_BASE_URL || 'http://localhost:8080',
        '/daily-share-card': env.VITE_API_BASE_URL || 'http://localhost:8080',
        '/questions': env.VITE_API_BASE_URL || 'http://localhost:8080',
        '/leaderboard': env.VITE_API_BASE_URL || 'http://localhost:8080',
        '/users': env.VITE_API_BASE_URL || 'http://localhost:8080',
        '/admin': env.VITE_API_BASE_URL || 'http://localhost:8080',
        '/health': env.VITE_API_BASE_URL || 'http://localhost:8080',
        '/metrics': env.VITE_API_BASE_URL || 'http://localhost:8080',
        '/ws': {
          target: env.VITE_API_BASE_URL || 'http://localhost:8080',
          ws: true,
        },
      },
    },
  }
})
