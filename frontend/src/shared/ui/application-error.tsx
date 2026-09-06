import { Component, type PropsWithChildren } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from './button'

export function ApplicationError() {
  const { t } = useTranslation()
  return (
    <main role="alert" className="grid min-h-screen place-content-center gap-5 p-6 text-center">
      <h1 className="text-xl font-bold">{t('errors.generic')}</h1>
      <Button onClick={() => window.location.reload()}>{t('common.retry')}</Button>
    </main>
  )
}

export class ApplicationErrorBoundary extends Component<PropsWithChildren, { failed: boolean }> {
  state = { failed: false }
  static getDerivedStateFromError() {
    return { failed: true }
  }
  render() {
    return this.state.failed ? <ApplicationError /> : this.props.children
  }
}
