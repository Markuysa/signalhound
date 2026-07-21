import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Build to ui/dist, which the Go binary embeds via embed.FS (ADR-001). The dev server
// proxies /api to the Go process so the UI and backend are developed independently.
export default defineConfig({
  plugins: [react()],
  build: { outDir: 'dist' },
  server: {
    proxy: {
      '/api': { target: 'http://127.0.0.1:8080', changeOrigin: true },
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    globals: true,
  },
});
