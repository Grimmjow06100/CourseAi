import { Highlight, themes } from 'prism-react-renderer'
import { useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import ReactMarkdown from 'react-markdown'
import rehypeSanitize from 'rehype-sanitize'
import remarkGfm from 'remark-gfm'

function MermaidDiagram({ source }: { source: string }) {
  const { t } = useTranslation()
  const reactId = useId().replaceAll(':', '')
  const [svg, setSvg] = useState<string | null>(null)
  const [invalid, setInvalid] = useState(false)
  useEffect(() => {
    let active = true
    async function renderDiagram() {
      try {
        const { default: mermaid } = await import('mermaid')
        mermaid.initialize({
          startOnLoad: false,
          securityLevel: 'strict',
          theme: 'neutral',
          fontFamily: 'Manrope Variable, sans-serif',
        })
        const result = await mermaid.render(`course-ai-${reactId}`, source)
        if (active) setSvg(result.svg)
      } catch {
        if (active) setInvalid(true)
      }
    }
    void renderDiagram()
    return () => {
      active = false
    }
  }, [reactId, source])
  if (invalid)
    return (
      <pre>
        <code>{source}</code>
      </pre>
    )
  if (!svg) return <div className="h-32 animate-pulse rounded-md bg-muted" aria-label={t('common.loading')} />
  return (
    <div
      className="my-6 overflow-x-auto rounded-md border border-border bg-white p-4"
      dangerouslySetInnerHTML={{ __html: svg }}
    />
  )
}

function CodeBlock({ className, children }: { className?: string | undefined; children?: React.ReactNode }) {
  const code = Array.isArray(children)
    ? children
        .filter((child): child is string | number => typeof child === 'string' || typeof child === 'number')
        .join('')
    : typeof children === 'string' || typeof children === 'number'
      ? String(children)
      : ''
  const language = className?.replace('language-', '') ?? ''
  if (language === 'mermaid') return <MermaidDiagram key={code} source={code} />
  if (!className) return <code>{children}</code>
  return (
    <Highlight theme={themes.vsDark} code={code} language={language || 'text'}>
      {({ className: rootClassName, style, tokens, getLineProps, getTokenProps }) => (
        <pre className={rootClassName} style={style}>
          {tokens.map((line, lineIndex) => (
            <div key={lineIndex} {...getLineProps({ line })}>
              {line.map((token, tokenIndex) => (
                <span key={tokenIndex} {...getTokenProps({ token })} />
              ))}
            </div>
          ))}
        </pre>
      )}
    </Highlight>
  )
}

export function Markdown({ children }: { children: string }) {
  return (
    <div className="markdown">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeSanitize]}
        components={{
          code: CodeBlock,
          pre: ({ children }) => <div className="markdown-code">{children}</div>,
          table: ({ children }) => (
            <div className="overflow-x-auto">
              <table>{children}</table>
            </div>
          ),
        }}
      >
        {children}
      </ReactMarkdown>
    </div>
  )
}
