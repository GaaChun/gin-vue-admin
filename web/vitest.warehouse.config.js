import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) }
  },
  test: {
    // Isolate this suite so warehouse tests cannot accidentally execute unrelated frontend tests.
    include: ['src/plugin/warehouse/**/*.test.{js,ts}'],
    // happy-dom supplies Vue's DOM APIs headlessly; browser runners are deliberately not configured.
    environment: 'happy-dom',
    setupFiles: ['./vitest.warehouse.setup.js'],
  },
})
