import * as Dialog from '@radix-ui/react-dialog'
import { UserButton } from '@clerk/react'
import { Link, Outlet, useRouterState } from '@tanstack/react-router'
import { BookOpenCheck, FolderClock, GraduationCap, LayoutDashboard, Menu, Plus, X } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/shared/lib/cn'
import { Button } from '@/shared/ui/button'

const navItems = [
  { to: '/', label: 'nav.dashboard', icon: LayoutDashboard },
  { to: '/generate', label: 'nav.generate', icon: Plus },
  { to: '/generations', label: 'nav.generations', icon: FolderClock },
  { to: '/courses', label: 'nav.courses', icon: GraduationCap },
] as const

function Navigation({ onNavigate }: { onNavigate?: () => void }) {
  const { t } = useTranslation()
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  return (
    <nav className="space-y-1" aria-label={t('common.menu')}>
      {navItems.map(({ to, label, icon: Icon }) => {
        const active = to === '/' ? pathname === '/' : pathname.startsWith(to)
        return (
          <Link
            key={to}
            to={to}
            onClick={onNavigate}
            className={cn(
              'flex h-10 items-center gap-3 rounded-md px-3 text-sm font-semibold text-muted-foreground transition-colors hover:bg-muted hover:text-foreground',
              active && 'bg-surface text-primary shadow-sm',
            )}
          >
            <Icon className="size-4" aria-hidden="true" />
            {t(label)}
          </Link>
        )
      })}
    </nav>
  )
}

function LanguageControl() {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage?.startsWith('en') ? 'en' : 'fr'
  return (
    <div className="flex rounded-md border border-border bg-surface p-0.5" aria-label={t('common.language')}>
      {(['fr', 'en'] as const).map((value) => (
        <button
          key={value}
          type="button"
          onClick={() => void i18n.changeLanguage(value)}
          className={cn(
            'h-7 rounded px-2 text-xs font-bold uppercase text-muted-foreground',
            value === language && 'bg-primary text-primary-foreground',
          )}
          aria-pressed={value === language}
        >
          {value}
        </button>
      ))}
    </div>
  )
}

export function AppShell() {
  const { t } = useTranslation()
  const [menuOpen, setMenuOpen] = useState(false)
  return (
    <div className="min-h-screen bg-background lg:grid lg:grid-cols-[248px_minmax(0,1fr)]">
      <aside className="fixed inset-y-0 left-0 hidden w-[248px] border-r border-border bg-sidebar p-4 lg:flex lg:flex-col">
        <Link to="/" className="mb-8 flex items-center gap-3 px-2 py-2 font-extrabold">
          <span className="grid size-9 place-items-center rounded-md bg-primary text-primary-foreground">
            <BookOpenCheck className="size-5" />
          </span>
          <span>{t('brand.name')}</span>
        </Link>
        <Navigation />
        <p className="mt-auto px-3 pb-2 text-xs leading-5 text-muted-foreground">{t('brand.tagline')}</p>
      </aside>
      <div className="min-w-0 lg:col-start-2">
        <header className="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-border bg-background/95 px-4 backdrop-blur lg:px-8">
          <div className="lg:hidden">
            <Dialog.Root open={menuOpen} onOpenChange={setMenuOpen}>
              <Dialog.Trigger asChild>
                <Button variant="icon" aria-label={t('common.menu')}>
                  <Menu className="size-5" />
                </Button>
              </Dialog.Trigger>
              <Dialog.Portal>
                <Dialog.Overlay className="fixed inset-0 z-40 bg-black/40" />
                <Dialog.Content
                  aria-describedby={undefined}
                  className="fixed inset-y-0 left-0 z-50 w-[min(84vw,300px)] border-r border-border bg-sidebar p-4"
                >
                  <Dialog.Title className="sr-only">{t('common.menu')}</Dialog.Title>
                  <div className="mb-8 flex items-center justify-between">
                    <span className="flex items-center gap-2 font-extrabold">
                      <BookOpenCheck className="size-5 text-primary" />
                      {t('brand.name')}
                    </span>
                    <Dialog.Close asChild>
                      <Button variant="icon" aria-label={t('common.close')}>
                        <X className="size-5" />
                      </Button>
                    </Dialog.Close>
                  </div>
                  <Navigation onNavigate={() => setMenuOpen(false)} />
                </Dialog.Content>
              </Dialog.Portal>
            </Dialog.Root>
          </div>
          <div className="hidden text-sm font-semibold text-muted-foreground lg:block">
            {t('brand.tagline')}
          </div>
          <div className="ml-auto flex items-center gap-3">
            <LanguageControl />
            {import.meta.env.DEV && import.meta.env.VITE_E2E_MODE === 'true' ? (
              <span
                className="grid size-8 place-items-center rounded-full bg-primary text-xs font-bold text-primary-foreground"
                aria-label="Test user"
              >
                TU
              </span>
            ) : (
              <UserButton />
            )}
          </div>
        </header>
        <main className="mx-auto w-full max-w-[1440px] px-4 py-6 sm:px-6 lg:px-8 lg:py-8">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
