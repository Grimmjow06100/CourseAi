import { createRootRouteWithContext, Link, Outlet } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { ApplicationError } from '@/shared/ui/application-error'
import { Toaster } from 'sonner'
import type { AppRouterContext } from '@/app/router-context'

function NotFoundPage() {
  const { t } = useTranslation()
  return (
    <main className="grid min-h-screen place-items-center p-6 text-center">
      <div>
        <p className="font-mono text-sm text-primary">404</p>
        <h1 className="mt-3 text-2xl font-bold">{t('notFound.title')}</h1>
        <Link to="/" className="mt-6 inline-block font-semibold text-primary underline">
          {t('notFound.action')}
        </Link>
      </div>
    </main>
  )
}

export const Route = createRootRouteWithContext<AppRouterContext>()({
  component: () => (
    <>
      <Outlet />
      <Toaster
        position="top-right"
        closeButton
        toastOptions={{
          style: {
            background: 'var(--surface)',
            color: 'var(--foreground)',
            border: '1px solid var(--border)',
          },
        }}
      />
    </>
  ),
  notFoundComponent: NotFoundPage,
  errorComponent: ApplicationError,
})
