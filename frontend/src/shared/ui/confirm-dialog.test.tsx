import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import '@/shared/i18n'
import { ConfirmDialog } from './confirm-dialog'

it('keeps the confirmation open on rejection and closes it only after success', async () => {
  const onConfirm = vi
    .fn<() => Promise<void>>()
    .mockRejectedValueOnce(new Error('offline'))
    .mockResolvedValueOnce(undefined)
  render(
    <ConfirmDialog
      trigger={<button>Remove</button>}
      title="Delete course?"
      description="Permanent deletion"
      confirmLabel="Confirm"
      cancelLabel="Cancel"
      onConfirm={onConfirm}
    />,
  )
  fireEvent.click(screen.getByText('Remove'))
  fireEvent.click(screen.getByText('Confirm'))
  expect(await screen.findByRole('alert')).toBeVisible()
  expect(screen.getByRole('alertdialog')).toBeVisible()
  fireEvent.click(screen.getByText('Confirm'))
  await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument())
  expect(onConfirm).toHaveBeenCalledTimes(2)
})
