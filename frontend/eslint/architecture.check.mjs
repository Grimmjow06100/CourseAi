import assert from 'node:assert/strict'
import test from 'node:test'
import { Linter } from 'eslint'
import architecture from './architecture.mjs'

function violations(filename, code) {
  return new Linter().verify(
    code,
    {
      plugins: { local: { rules: { architecture } } },
      rules: { 'local/architecture': 'error' },
    },
    { filename: `${process.cwd()}/src/${filename}` },
  )
}

test('shared cannot reach features through aliases, relative imports or re-exports', () => {
  for (const code of [
    "import x from '@/features/catalog/api'",
    "import x from '../../features/catalog/api'",
    "export * from '../../features/catalog/api'",
    "import('../../features/catalog/api')",
  ])
    assert.equal(violations('shared/api/example.js', code).length, 1, code)
})

test('features have a directed dependency graph, including relative imports', () => {
  assert.equal(violations('features/catalog/example.js', "import x from '../generation/api'").length, 0)
  assert.equal(violations('features/generation/example.js', "import x from '../catalog/api'").length, 1)
  assert.equal(violations('features/generation/example.js', "import x from '@/routes/index'").length, 1)
  assert.equal(
    violations('features/generation/example.js', "import x from '@/shared/api/query-keys'").length,
    0,
  )
})

test('composition and UI primitives keep their respective responsibilities', () => {
  assert.equal(violations('routes/example.js', "import x from '@/features/catalog/api'").length, 0)
  assert.equal(violations('components/ui/example.js', "import x from '@/features/catalog/api'").length, 1)
})
