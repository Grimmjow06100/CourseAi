import { createRootRouteWithContext, Link, Outlet } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { ApplicationError } from '@/shared/ui/application-error'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Button } from '@/components/ui/button'
import type { AppRouterContext } from '@/app/router-context'

function NotFoundPage() {
  const { t } = useTranslation()
  return (
    <main className="grid min-h-screen place-items-center p-6 text-center">
      <div>
        <p className="font-mono text-sm text-primary">404</p>
        <h1 className="mt-3 text-2xl font-bold">{t('notFound.title')}</h1>
        <p className="mt-3 text-sm text-muted-foreground">{t('notFound.description')}</p>
        <Button asChild className="mt-6">
          <Link to="/">{t('notFound.action')}</Link>
        </Button>
      </div>
    </main>
  )
}

export const Route = createRootRouteWithContext<AppRouterContext>()({
  component: () => (
    <TooltipProvider>
      <Outlet />
    </TooltipProvider>
  ),
  notFoundComponent: NotFoundPage,
  errorComponent: ApplicationError,
})
