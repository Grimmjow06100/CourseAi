import { redirect } from '@tanstack/react-router'

export function requireAuthentication(isSignedIn: boolean) {
  if (!isSignedIn) throw redirect({ to: '/sign-in' })
}
