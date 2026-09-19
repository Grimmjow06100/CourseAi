import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import '@/shared/i18n'
import { GenerationForm } from './generation-form'

const { mutateAsync, navigate } = vi.hoisted(() => ({ mutateAsync: vi.fn(), navigate: vi.fn() }))
vi.mock('@tanstack/react-router', () => ({ useNavigate: () => navigate }))
vi.mock('./api', () => ({
  useStartGeneration: () => ({ mutateAsync, isPending: false, error: null }),
}))

beforeEach(() => {
  mutateAsync.mockReset()
  navigate.mockReset()
})

it('opens the accepted request after the form submission succeeds', async () => {
  mutateAsync.mockResolvedValue({ requestId: 'accepted-request' })
  render(<GenerationForm />)
  fireEvent.change(screen.getByRole('textbox'), { target: { value: 'Learn Linux administration' } })
  fireEvent.click(screen.getByRole('button', { name: /générer la formation|generate course/i }))
  await waitFor(() =>
    expect(navigate).toHaveBeenCalledWith({
      to: '/generations/$requestId',
      params: { requestId: 'accepted-request' },
    }),
  )
})

it('reuses an idempotency key only for the same normalized prompt after an uncertain response', async () => {
  mutateAsync.mockRejectedValue(new Error('network'))
  render(<GenerationForm />)
  const prompt = screen.getByRole('textbox')
  const submit = screen.getByRole('button', { name: /générer la formation|generate course/i })
  fireEvent.change(prompt, { target: { value: 'Learn Linux administration' } })
  fireEvent.click(submit)
  await waitFor(() => expect(mutateAsync).toHaveBeenCalledTimes(1))
  const first: unknown = mutateAsync.mock.calls[0]?.[0]
  fireEvent.click(submit)
  await waitFor(() => expect(mutateAsync).toHaveBeenCalledTimes(2))
  expect(mutateAsync.mock.calls[1]?.[0]).toEqual(first)
  fireEvent.change(prompt, { target: { value: 'Learn React development' } })
  fireEvent.click(submit)
  await waitFor(() => expect(mutateAsync).toHaveBeenCalledTimes(3))
  expect(mutateAsync.mock.calls[2]?.[0]).not.toMatchObject(first as object)
  const calls = mutateAsync.mock.calls as [{ prompt: string; idempotencyKey: string }][]
  expect(calls[2]?.[0].idempotencyKey).not.toBe(calls[0]?.[0].idempotencyKey)
})
