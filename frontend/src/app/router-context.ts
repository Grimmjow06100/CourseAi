export interface RouterAuthContext {
  isSignedIn: boolean
}

export interface AppRouterContext {
  auth: RouterAuthContext
}
