import { ClerkProvider, useAuth } from '@clerk/react'
import { enUS, frFR } from '@clerk/localizations'
import { QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { RouterProvider } from '@tanstack/react-router'
import { useCallback, useLayoutEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ApiClientProvider } from '@/shared/api/context'
import type { TokenProvider } from '@/shared/api/client'
import { loadEnvironment, type Environment } from '@/shared/config/environment'
import { LoadingState } from '@/shared/ui/feedback'
import { createQueryClient } from './query-client'
import { createAppRouter } from './router'
import { SessionLifetime } from './session-lifetime'
import { useTheme } from '@/shared/hooks/use-theme'

function Runtime({
  getToken,
  isSignedIn,
  environment,
}: {
  getToken: TokenProvider
  isSignedIn: boolean
  environment: Environment
}) {
  const [resources] = useState(() => ({
    queryClient: createQueryClient(),
    router: createAppRouter(isSignedIn),
    lifetime: new SessionLifetime(),
  }))
  useLayoutEffect(() => {
    resources.lifetime.activate()
    return () => {
      resources.lifetime.dispose()
      resources.queryClient.clear()
    }
  }, [resources])
  return (
    <QueryClientProvider client={resources.queryClient}>
      <ApiClientProvider
        baseUrl={environment.VITE_API_BASE_URL}
        getToken={getToken}
        getSessionSignal={resources.lifetime.getSignal}
      >
        <RouterProvider router={resources.router} />
      </ApiClientProvider>
      {import.meta.env.DEV && import.meta.env.VITE_E2E_MODE !== 'true' ? (
        <ReactQueryDevtools initialIsOpen={false} />
      ) : null}
    </QueryClientProvider>
  )
}

function ClerkRuntime({ environment }: { environment: Environment }) {
  const { isLoaded, isSignedIn, getToken, userId, sessionId } = useAuth()
  const { t } = useTranslation()
  const tokenProvider = useCallback<TokenProvider>((options) => getToken(options), [getToken])
  if (!isLoaded) return <LoadingState label={t('common.loading')} />
  return (
    <Runtime
      key={`${userId ?? 'anonymous'}:${sessionId ?? 'none'}`}
      environment={environment}
      getToken={tokenProvider}
      isSignedIn={isSignedIn}
    />
  )
}

export function Application() {
  const { i18n } = useTranslation()
  const { resolvedTheme } = useTheme()
  const tokens = getComputedStyle(document.documentElement)
  const environment = loadEnvironment(import.meta.env, { production: import.meta.env.PROD })
  const localization = i18n.resolvedLanguage?.startsWith('en') ? enUS : frFR
  if (import.meta.env.DEV && environment.VITE_E2E_MODE) {
    return <Runtime environment={environment} getToken={() => Promise.resolve('e2e-test-token')} isSignedIn />
  }
  return (
    <ClerkProvider
      publishableKey={environment.VITE_CLERK_PUBLISHABLE_KEY}
      localization={localization}
      signInUrl="/sign-in"
      signUpUrl="/sign-up"
      afterSignOutUrl="/sign-in"
      appearance={{
        variables: {
          colorPrimary: tokens.getPropertyValue('--primary').trim(),
          colorBackground: tokens.getPropertyValue('--card').trim(),
          colorForeground: tokens.getPropertyValue('--foreground').trim(),
          colorMutedForeground: tokens.getPropertyValue('--muted-foreground').trim(),
          colorDanger: tokens.getPropertyValue('--destructive').trim(),
          borderRadius: '8px',
          fontFamily: 'Geist Variable, sans-serif',
        },
        elements: {
          rootBox: 'w-full',
          cardBox: 'w-full shadow-none',
          card: 'shadow-none border border-border',
          userButtonPopoverCard: resolvedTheme === 'dark' ? 'dark' : '',
        },
      }}
    >
      <ClerkRuntime environment={environment} />
    </ClerkProvider>
  )
}
