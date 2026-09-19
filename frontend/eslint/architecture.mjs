import path from 'node:path'

// Dependencies between features are deliberately directed: reader -> catalog -> generation.
const featureDependencies = {
  auth: [],
  generation: [],
  catalog: ['generation'],
  'course-reader': ['catalog', 'generation'],
}

export default {
  meta: {
    type: 'problem',
    schema: [],
    messages: { boundary: '{{source}} must not depend on {{target}}.' },
  },
  create(context) {
    const filename = context.filename.replaceAll('\\', '/')
    const rootIndex = filename.lastIndexOf('/src/')
    if (rootIndex < 0) return {}
    const root = filename.slice(0, rootIndex + 5)
    const source = filename.slice(root.length)

    function check(node) {
      const specifier = node.source?.value
      if (typeof specifier !== 'string') return
      const target = specifier.startsWith('@/')
        ? specifier.slice(2)
        : specifier.startsWith('.')
          ? path.posix.relative(
              root,
              path.posix.normalize(path.posix.join(path.posix.dirname(filename), specifier)),
            )
          : null
      if (target === null) return

      const [layer, feature] = source.split('/')
      const [targetLayer, targetFeature] = target.split('/')
      let forbidden = false
      if (layer === 'shared' || (layer === 'components' && feature === 'ui')) {
        forbidden = ['app', 'routes', 'features'].includes(targetLayer)
      }
      if (layer === 'features') {
        forbidden = ['app', 'routes'].includes(targetLayer)
        if (targetLayer === 'features' && targetFeature !== feature) {
          forbidden = !featureDependencies[feature]?.includes(targetFeature)
        }
      }
      if (forbidden) context.report({ node, messageId: 'boundary', data: { source, target } })
    }

    return {
      ImportDeclaration: check,
      ExportNamedDeclaration: check,
      ExportAllDeclaration: check,
      ImportExpression: check,
    }
  },
}
