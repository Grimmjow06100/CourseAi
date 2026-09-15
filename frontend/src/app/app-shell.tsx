import * as Dialog from '@radix-ui/react-dialog'
import { UserButton } from '@clerk/react'
import { Link, Outlet, useRouterState } from '@tanstack/react-router'
import {
  ArrowUpRight,
  BookOpenCheck,
  ChevronRight,
  FolderClock,
  GraduationCap,
  LayoutDashboard,
  Menu,
  Plus,
  Sparkles,
  X,
} from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/shared/lib/cn'
import { Button } from '@/shared/ui/button'
import { ThemeControl } from '@/shared/ui/theme-control'

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
              'flex min-h-12 items-center gap-3 rounded-xl px-3 text-sm font-semibold text-muted-foreground transition-colors hover:bg-muted hover:text-foreground',
              active && 'bg-success-soft text-primary',
            )}
            aria-current={active ? 'page' : undefined}
          >
            <Icon className="size-4" aria-hidden="true" />
            {t(label)}
            {active ? <span className="ml-auto size-1.5 rounded-full bg-primary" aria-hidden="true" /> : null}
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
    <div
      role="group"
      className="flex rounded-xl border border-border bg-surface p-1"
      aria-label={t('common.language')}
    >
      {(['fr', 'en'] as const).map((value) => (
        <button
          key={value}
          type="button"
          onClick={() => void i18n.changeLanguage(value)}
          className={cn(
            'min-h-8 min-w-8 rounded-lg px-2 text-xs font-bold uppercase text-muted-foreground',
            value === language && 'bg-success-soft text-primary',
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
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  const currentPage = navItems.find(({ to }) => to !== '/' && pathname.startsWith(to)) ?? navItems[0]
  return (
    <div className="min-h-screen bg-background lg:grid lg:grid-cols-[248px_minmax(0,1fr)]">
      <a
        href="#main-content"
        className="fixed left-4 top-3 z-[60] -translate-y-24 rounded-xl bg-primary px-4 py-3 font-bold text-primary-foreground focus:translate-y-0"
      >
        {t('design.skip')}
      </a>
      <aside className="fixed inset-y-0 left-0 hidden w-[248px] overflow-y-auto border-r border-border bg-sidebar px-5 py-7 lg:flex lg:flex-col">
        <Link to="/" className="mb-10 flex items-center gap-3 px-2 text-xl font-extrabold tracking-tight">
          <span className="grid size-10 place-items-center rounded-xl bg-primary text-primary-foreground">
            <BookOpenCheck className="size-5" />
          </span>
          <span>{t('brand.name')}</span>
        </Link>
        <p className="mb-3 px-3 text-[10px] font-extrabold tracking-[.16em] text-muted-foreground">
          {t('design.navigation')}
        </p>
        <Navigation />
        <div className="mt-auto pt-12">
          <div className="rounded-2xl border border-border bg-background p-4">
            <Sparkles className="mb-3 size-5 text-primary" aria-hidden="true" />
            <p className="text-sm font-extrabold leading-6">{t('design.sidebarTitle')}</p>
            <p className="mt-2 text-xs leading-5 text-muted-foreground">{t('design.sidebarText')}</p>
            <Link
              to="/generate"
              className="mt-4 flex min-h-9 items-center justify-between text-xs font-bold text-primary"
            >
              {t('nav.generate')}
              <ArrowUpRight className="size-4" />
            </Link>
          </div>
          <p className="mt-5 px-1 text-[11px] leading-5 text-muted-foreground">{t('brand.tagline')}</p>
        </div>
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
                  className="fixed inset-y-0 left-0 z-50 w-[min(84vw,300px)] overflow-y-auto border-r border-border bg-sidebar p-4"
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
          <div className="hidden items-center gap-3 text-xs font-semibold text-muted-foreground lg:flex">
            {t('design.workspace')}
            <ChevronRight className="size-3" aria-hidden="true" />
            <span className="text-foreground">{t(currentPage.label)}</span>
          </div>
          <div className="ml-auto flex items-center gap-3">
            <ThemeControl />
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
        <main
          id="main-content"
          tabIndex={-1}
          className="mx-auto w-full max-w-[1440px] px-4 py-6 outline-none sm:px-6 lg:px-8 lg:py-8"
        >
          <Outlet />
        </main>
      </div>
    </div>
  )
}
