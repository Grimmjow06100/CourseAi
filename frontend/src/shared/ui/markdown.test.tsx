import { render, screen } from '@testing-library/react'
import { Markdown } from './markdown'

it('renders code without nested pre elements and keeps raw HTML inert', () => {
  const { container } = render(
    <Markdown>
      {'```js\nconst x = 1\n```\n\n<script>alert(1)</script>\n\n[unsafe](javascript:alert(1))'}
    </Markdown>,
  )
  expect(container.querySelector('pre pre')).toBeNull()
  expect(container.querySelector('pre')).not.toBeNull()
  expect(container.querySelector('script')).toBeNull()
  expect(screen.getByText('unsafe')).not.toHaveAttribute('href', 'javascript:alert(1)')
})
