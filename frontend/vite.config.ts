import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')

  // Proxy target: used when frontend runs in Docker (VITE_PROXY_TARGET=http://core:8080).
  // Client always uses relative URLs so fetch goes to same origin; proxy forwards to backend.
  const proxyTarget = env.VITE_PROXY_TARGET || env.VITE_API_BASE_URL || 'http://localhost:8080'

  return {
    plugins: [react()],
    server: {
      port: 5173,
      allowedHosts: mode === 'development' ? true : ['localhost'],
      proxy: {
        '/join': proxyTarget,
        '/rooms': proxyTarget,
        '/daily-challenge': proxyTarget,
        '/daily-submit': proxyTarget,
        '/daily-share-card': proxyTarget,
        '/health': proxyTarget,
        '/metrics': proxyTarget,
        '/api/tutor': proxyTarget,
        '/ws': {
          target: proxyTarget,
          ws: true,
        },
      },
    },
  }
})
