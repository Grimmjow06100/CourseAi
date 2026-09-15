import { fireEvent, render, screen } from '@testing-library/react'
import { i18n } from '@/shared/i18n'
import { ApiError } from '@/shared/api/errors'
import { ApiErrorNotice } from './api-error-notice'

beforeEach(async () => {
  await i18n.changeLanguage('fr')
})

it('offers retry and sign-in without calling a rejected API token an expired session', () => {
  const onRetry = vi.fn()
  render(
    <ApiErrorNotice
      error={new ApiError('Invalid token', 401, 'unauthenticated', 'request-123')}
      onRetry={onRetry}
    />,
  )
  expect(screen.getByRole('alert')).toHaveTextContent('L’API n’a pas pu valider votre connexion')
  expect(screen.getByRole('alert')).not.toHaveTextContent('Votre session a expiré')
  expect(screen.getByRole('alert')).toHaveTextContent('request-123')
  expect(screen.getByRole('link', { name: 'Se connecter' })).toHaveAttribute('href', '/sign-in')
  fireEvent.click(screen.getByRole('button', { name: 'Réessayer' }))
  expect(onRetry).toHaveBeenCalledOnce()
})

it('asks for sign-in when Clerk no longer has a session', () => {
  render(
    <ApiErrorNotice error={new ApiError('No session', 401, 'session_expired', null)} onRetry={vi.fn()} />,
  )
  expect(screen.getByRole('alert')).toHaveTextContent('Votre session a expiré')
  expect(screen.getByRole('link', { name: 'Se connecter' })).toBeVisible()
  expect(screen.queryByRole('button', { name: 'Réessayer' })).not.toBeInTheDocument()
})
