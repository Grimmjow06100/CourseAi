import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'
import architecture from './eslint/architecture.mjs'

export default defineConfig([
  globalIgnores([
    'dist',
    'coverage',
    'playwright-report',
    'src/routeTree.gen.ts',
    'src/shared/api/schema.gen.ts',
  ]),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.strictTypeChecked,
      tseslint.configs.stylisticTypeChecked,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      globals: globals.browser,
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    rules: {
      '@typescript-eslint/no-confusing-void-expression': 'off',
      '@typescript-eslint/restrict-template-expressions': ['error', { allowNumber: true }],
    },
  },
  {
    files: ['src/components/ui/*.{ts,tsx}'],
    rules: {
      'react-refresh/only-export-components': [
        'error',
        { allowExportNames: ['buttonVariants', 'badgeVariants', 'tabsListVariants', 'useSidebar'] },
      ],
    },
  },
  {
    files: ['src/routes/**/*.{ts,tsx}'],
    rules: {
      'react-refresh/only-export-components': 'off',
      '@typescript-eslint/only-throw-error': 'off',
    },
  },
  {
    files: ['src/shared/api/context.tsx'],
    rules: { 'react-refresh/only-export-components': 'off' },
  },
  {
    files: ['src/features/auth/guard.ts'],
    rules: { '@typescript-eslint/only-throw-error': 'off' },
  },
  {
    files: ['src/**/*.{ts,tsx}'],
    plugins: { local: { rules: { architecture } } },
    rules: {
      'local/architecture': 'error',
    },
  },
])
