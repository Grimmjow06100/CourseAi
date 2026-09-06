import { createRouter } from '@tanstack/react-router'
import { routeTree } from '@/routeTree.gen'
import type { AppRouterContext } from './router-context'

export const createAppRouter = (isSignedIn: boolean) =>
  createRouter({
    routeTree,
    context: { auth: { isSignedIn } } satisfies AppRouterContext,
    defaultPreload: 'intent',
    defaultPreloadStaleTime: 0,
    scrollRestoration: true,
  })

declare module '@tanstack/react-router' {
  interface Register {
    router: ReturnType<typeof createAppRouter>
  }
}
