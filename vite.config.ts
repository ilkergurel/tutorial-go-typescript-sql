import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { fileURLToPath, URL } from 'node:url';

const apiTarget = process.env.STACKLAB_API_PROXY_TARGET ?? 'http://127.0.0.1:8080';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  optimizeDeps: {
    exclude: ['lucide-react', 'yaml'],
  },
  // The dev server proxies backend calls so the browser stays same-origin (the
  // backend sends no CORS headers). /healthz is the frontend's backend-or-
  // localStorage probe. In Compose the backend is reached by service name
  // (STACKLAB_API_PROXY_TARGET); a host-run `npm run dev` uses the published port.
  server: {
    proxy: {
      '/api': apiTarget,
      '/healthz': apiTarget,
    },
  },
});
