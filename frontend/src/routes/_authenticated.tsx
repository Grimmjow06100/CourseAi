import { createFileRoute } from '@tanstack/react-router'
import { AppShell } from '@/app/app-shell'
import { requireAuthentication } from '@/features/auth/guard'

export const Route = createFileRoute('/_authenticated')({
  beforeLoad: ({ context }) => {
    requireAuthentication(context.auth.isSignedIn)
  },
  component: AppShell,
})
