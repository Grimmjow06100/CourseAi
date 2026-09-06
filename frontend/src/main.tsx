import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { ApplicationError, ApplicationErrorBoundary } from '@/shared/ui/application-error'
import { Application } from '@/app/application'
import '@/shared/i18n'
import './index.css'

const rootElement = document.getElementById('root')
if (!rootElement) throw new Error('Application root element not found')

const root = createRoot(rootElement)
window.addEventListener('vite:preloadError', (event) => {
  event.preventDefault()
  root.render(<ApplicationError />)
})
root.render(
  <StrictMode>
    <ApplicationErrorBoundary>
      <Application />
    </ApplicationErrorBoundary>
  </StrictMode>,
)
