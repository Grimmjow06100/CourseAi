import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { gzipSync } from 'node:zlib'

// Run after `vite build --manifest` for both revisions, using their lockfiles
// and identical public build variables. Count each static dependency once.
const [baselineDirectory, currentDirectory = 'dist'] = process.argv.slice(2)
if (!baselineDirectory)
  throw new Error('Usage: node scripts/compare-initial-bundles.mjs <baseline-dist> [current-dist]')

function measure(directory, roots) {
  const manifest = JSON.parse(readFileSync(resolve(directory, '.vite/manifest.json'), 'utf8'))
  const visited = new Set()
  const files = new Set()
  function visit(key) {
    if (visited.has(key)) return
    visited.add(key)
    const chunk = manifest[key]
    if (!chunk) throw new Error(`Missing manifest entry: ${key}`)
    if (chunk.file.endsWith('.js')) files.add(chunk.file)
    for (const dependency of chunk.imports ?? []) visit(dependency)
  }
  roots.forEach(visit)
  let bytes = 0
  let gzipBytes = 0
  for (const file of files) {
    const content = readFileSync(resolve(directory, file))
    bytes += content.length
    gzipBytes += gzipSync(content).length
  }
  return { files: files.size, bytes, gzipBytes }
}

const screens = {
  bootstrap: ['index.html'],
  authenticatedHome: [
    'index.html',
    'src/routes/_authenticated.tsx?tsr-split=component',
    'src/routes/_authenticated.index.tsx?tsr-split=component',
  ],
}
const result = Object.fromEntries(
  Object.entries(screens).map(([name, roots]) => {
    const baseline = measure(baselineDirectory, roots)
    const current = measure(currentDirectory, roots)
    return [
      name,
      {
        baseline,
        current,
        changePercent: Number(((current.bytes / baseline.bytes - 1) * 100).toFixed(2)),
        gzipChangePercent: Number(((current.gzipBytes / baseline.gzipBytes - 1) * 100).toFixed(2)),
      },
    ]
  }),
)
console.log(JSON.stringify(result, null, 2))
if (result.authenticatedHome.changePercent > 10) process.exitCode = 1
