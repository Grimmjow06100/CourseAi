import { act, cleanup, render, renderHook, screen } from '@testing-library/react'
import { focusManager, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { PropsWithChildren } from 'react'
import { courseKeys, generationKeys } from '@/shared/api/query-keys'
import { useGenerationList, useGenerationStatus } from './api'
import { useGenerationTracking } from './tracking-api'
import { GenerationTracker } from './generation-tracker'
import { trackingFixture } from '@/test/tracking-fixture'

vi.mock('@tanstack/react-router', () => ({
  Link: ({ children }: PropsWithChildren) => <a href="/courses/course">{children}</a>,
  useNavigate: () => vi.fn(),
}))
const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/context', () => ({ useApiClient: () => ({ GET: get }) }))
vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key, i18n: { language: 'fr' } }),
}))

function setup() {
  const cache = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: 30_000, refetchOnWindowFocus: false } },
  })
  function Wrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={cache}>{children}</QueryClientProvider>
  }
  return { cache, Wrapper }
}

function success(data: unknown) {
  return { data, response: new Response('{}', { status: 200 }) }
}

afterEach(() => {
  cleanup()
  vi.useRealTimers()
  get.mockReset()
  focusManager.setFocused(undefined)
})

it('offers access to completed content without claiming the pipeline succeeded', () => {
  const { cache, Wrapper } = setup()
  render(
    <GenerationTracker
      snapshot={trackingFixture({
        pipelineStatus: 'failed',
        contentComplete: true,
        reconciliation: 'pending',
      })}
    />,
    { wrapper: Wrapper },
  )
  expect(screen.getByText('tracking.repair')).toBeInTheDocument()
  expect(screen.queryByText('tracking.restart')).not.toBeInTheDocument()
  expect(screen.getByRole('link')).toBeInTheDocument()
  cache.clear()
})

it('refreshes a terminal status when window focus returns', async () => {
  vi.useFakeTimers()
  const { cache, Wrapper } = setup()
  get.mockResolvedValue(success({ pipelineStatus: 'failed', requestId: 'request' }))
  const hook = renderHook(() => useGenerationStatus('request'), {
    wrapper: Wrapper,
  })
  await act(async () => {
    await vi.advanceTimersByTimeAsync(10)
  })
  expect(hook.result.current.data?.pipelineStatus).toBe('failed')
  get.mockResolvedValue(success({ pipelineStatus: 'completed', requestId: 'request' }))
  await act(async () => {
    await vi.advanceTimersByTimeAsync(60_000)
    focusManager.setFocused(false)
    focusManager.setFocused(true)
    await vi.advanceTimersByTimeAsync(10)
  })
  expect(get).toHaveBeenCalledTimes(2)
  expect(hook.result.current.data?.pipelineStatus).toBe('completed')
  hook.unmount()
  cache.clear()
})

it('tracking changes invalidate both history and course content', async () => {
  vi.useFakeTimers()
  const { cache, Wrapper } = setup()
  cache.setQueryData(courseKeys.detail('course'), { id: 'course' })
  cache.setQueryData(generationKeys.list({ page: 1, pageSize: 20 }), {
    items: [],
  })
  get.mockResolvedValue(success(trackingFixture({ pipelineStatus: 'completed' })))
  const hook = renderHook(() => useGenerationTracking('request'), {
    wrapper: Wrapper,
  })
  await act(async () => {
    await vi.advanceTimersByTimeAsync(10)
  })
  expect(cache.getQueryState(courseKeys.detail('course'))?.isInvalidated).toBe(true)
  expect(cache.getQueryState(generationKeys.list({ page: 1, pageSize: 20 }))?.isInvalidated).toBe(true)
  hook.unmount()
  cache.clear()
})

it('polls mounted history until generation completes', async () => {
  vi.useFakeTimers()
  const { cache, Wrapper } = setup()
  get.mockResolvedValue(success({ items: [{ pipelineStatus: 'running' }], total: 1 }))
  const hook = renderHook(() => useGenerationList({ page: 1 }), {
    wrapper: Wrapper,
  })
  await act(async () => {
    await vi.advanceTimersByTimeAsync(10)
  })
  get.mockResolvedValue(success({ items: [{ pipelineStatus: 'completed' }], total: 1 }))
  await act(async () => {
    await vi.advanceTimersByTimeAsync(60_000)
  })
  expect(get).toHaveBeenCalledTimes(2)
  hook.unmount()
  cache.clear()
})
