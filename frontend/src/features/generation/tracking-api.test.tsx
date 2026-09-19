import { act, cleanup, renderHook } from '@testing-library/react'
import { focusManager, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { PropsWithChildren } from 'react'
import { newestTracking, useGenerationTracking, useRetryOperations } from './tracking-api'
import { trackingFixture, operationFixture } from '@/test/tracking-fixture'

const { get, post } = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }))
vi.mock('@/shared/api/context', () => ({ useApiClient: () => ({ GET: get, POST: post }) }))
function setup() {
  const cache = new QueryClient({
    defaultOptions: { queries: { retry: false, notifyOnChangeProps: 'all' }, mutations: { retry: false } },
  })
  function Wrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={cache}>{children}</QueryClientProvider>
  }
  return { cache, Wrapper }
}
const success = (data: unknown) => ({ data, response: new Response('{}', { status: 200 }) })
afterEach(() => {
  cleanup()
  vi.useRealTimers()
  get.mockReset()
  post.mockReset()
  focusManager.setFocused(undefined)
})

it('rejects a delayed older revision and an older generation attempt', () => {
  const current = trackingFixture({ revision: '9007199254740994', generationAttempt: 2 })
  expect(
    newestTracking(current, trackingFixture({ revision: '9007199254740993', generationAttempt: 2 })),
  ).toBe(current)
  expect(
    newestTracking(current, trackingFixture({ revision: '9007199254740995', generationAttempt: 1 })),
  ).toBe(current)
})

it('keeps usable progress through a transient error, then resumes polling', async () => {
  vi.useFakeTimers()
  focusManager.setFocused(true)
  const { cache, Wrapper } = setup()
  const snapshot = trackingFixture()
  get
    .mockResolvedValueOnce(success(snapshot))
    .mockRejectedValueOnce(new Error('offline'))
    .mockResolvedValue(success(trackingFixture({ revision: '13' })))
  const hook = renderHook(() => useGenerationTracking(snapshot.requestId), { wrapper: Wrapper })
  await act(async () => {
    await vi.advanceTimersByTimeAsync(20)
  })
  expect(hook.result.current.data?.revision).toBe('12')
  await act(async () => {
    await hook.result.current.refetch()
    await vi.advanceTimersByTimeAsync(20)
  })
  expect(hook.result.current.data?.revision).toBe('12')
  expect(hook.result.current.isError).toBe(true)
  await act(async () => {
    await vi.advanceTimersByTimeAsync(10_100)
  })
  expect(hook.result.current.data?.revision).toBe('13')
  hook.unmount()
  cache.clear()
})

it('reuses the idempotency key after a lost response and changes it for a new operation version', async () => {
  const { cache, Wrapper } = setup()
  post.mockRejectedValue(new Error('lost response'))
  const hook = renderHook(() => useRetryOperations(trackingFixture().requestId), { wrapper: Wrapper })
  const op = operationFixture()
  const selected = [{ jobId: op.id, operationVersion: 1 }]
  await act(async () => {
    await hook.result.current.mutateAsync(selected).catch(() => undefined)
  })
  await act(async () => {
    await hook.result.current.mutateAsync(selected).catch(() => undefined)
  })
  await act(async () => {
    await hook.result.current.mutateAsync([{ jobId: op.id, operationVersion: 2 }]).catch(() => undefined)
  })
  const calls = post.mock.calls as [string, { params: { header: { 'Idempotency-Key': string } } }][]
  expect(calls[0]?.[1].params.header['Idempotency-Key']).toBeTruthy()
  expect(calls[1]?.[1].params.header).toEqual(calls[0]?.[1].params.header)
  expect(calls[2]?.[1].params.header).not.toEqual(calls[0]?.[1].params.header)
  hook.unmount()
  cache.clear()
})
