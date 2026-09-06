import { createContext, useContext, useMemo, type PropsWithChildren } from 'react'
import { createCourseAIClient, type CourseAIClient, type TokenProvider } from './client'

const ApiClientContext = createContext<CourseAIClient | null>(null)

interface ApiClientProviderProps extends PropsWithChildren {
  baseUrl: string
  getToken: TokenProvider
  getSessionSignal?: () => AbortSignal
}

export function ApiClientProvider({ baseUrl, getToken, getSessionSignal, children }: ApiClientProviderProps) {
  const client = useMemo(
    () => createCourseAIClient(baseUrl, getToken, getSessionSignal),
    [baseUrl, getToken, getSessionSignal],
  )
  return <ApiClientContext.Provider value={client}>{children}</ApiClientContext.Provider>
}

export function useApiClient() {
  const client = useContext(ApiClientContext)
  if (!client) throw new Error('useApiClient must be used within ApiClientProvider')
  return client
}
