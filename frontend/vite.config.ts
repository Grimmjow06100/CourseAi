import { defineConfig, loadEnv } from 'vite'
import { loadEnvironment } from './src/shared/config/environment.ts'
import react, { reactCompilerPreset } from '@vitejs/plugin-react'
import babel from '@rolldown/plugin-babel'
import tailwindcss from '@tailwindcss/vite'
import { tanstackRouter } from '@tanstack/router-plugin/vite'
import { fileURLToPath, URL } from 'node:url'

// https://vite.dev/config/
export default defineConfig(({ command, mode }) => {
  if (command === 'build')
    loadEnvironment(
      { ...loadEnv(mode, process.cwd(), 'VITE_'), ...process.env },
      {
        production: true,
        requireLiveKey: process.env.VERCEL_ENV === 'production',
      },
    )
  return {
    plugins: [
      tanstackRouter({ target: 'react', autoCodeSplitting: true }),
      react(),
      babel({ presets: [reactCompilerPreset()] }),
      tailwindcss(),
    ],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
  }
})
