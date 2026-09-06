import { act, fireEvent, render, screen } from '@testing-library/react'
import { DebouncedSearch } from './debounced-search'

describe('debounced URL search', () => {
  afterEach(() => vi.useRealTimers())
  it('only submits the latest user edit', () => {
    vi.useFakeTimers()
    const onSearch = vi.fn()
    render(<DebouncedSearch navigationKey="initial" initialValue="" label="Search" onSearch={onSearch} />)
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'linux' } })
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'react' } })
    void act(() => vi.advanceTimersByTime(350))
    expect(onSearch).toHaveBeenCalledExactlyOnceWith('react')
  })
  it('cancels pending edits when history navigation replaces URL state', () => {
    vi.useFakeTimers()
    const onSearch = vi.fn()
    const { rerender } = render(
      <DebouncedSearch navigationKey="linux" initialValue="linux" label="Search" onSearch={onSearch} />,
    )
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'stale edit' } })
    screen.getByRole('textbox').focus()
    rerender(<DebouncedSearch navigationKey="go" initialValue="go" label="Search" onSearch={onSearch} />)
    void act(() => vi.advanceTimersByTime(500))
    expect(screen.getByRole('textbox')).toHaveValue('go')
    expect(screen.getByRole('textbox')).toHaveFocus()
    expect(onSearch).not.toHaveBeenCalled()
  })
})
