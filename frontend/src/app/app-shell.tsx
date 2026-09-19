import { UserButton } from '@clerk/react'
import { Link, Outlet, useRouterState } from '@tanstack/react-router'
import { BookOpen, FolderClock, Home, Languages, PanelLeftClose, Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarSeparator,
  SidebarTrigger,
  useSidebar,
} from '@/components/ui/sidebar'
import { CourseAiLogo } from '@/shared/ui/course-ai-logo'
import { ThemeControl } from '@/shared/ui/theme-control'

const navItems = [
  { to: '/', label: 'nav.dashboard', icon: Home },
  { to: '/generate', label: 'nav.generate', icon: Plus },
  { to: '/generations', label: 'nav.generations', icon: FolderClock },
  { to: '/courses', label: 'nav.courses', icon: BookOpen },
] as const

function LanguageControl() {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage?.startsWith('en') ? 'en' : 'fr'
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" aria-label={t('common.language')} title={t('common.language')}>
          <Languages className="size-4" aria-hidden="true" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent side="top" align="start">
        <DropdownMenuRadioGroup value={language} onValueChange={(value) => void i18n.changeLanguage(value)}>
          <DropdownMenuRadioItem value="fr">Français</DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="en">English</DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function WorkspaceNavigation() {
  const { t } = useTranslation()
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  const { state, isMobile, setOpenMobile } = useSidebar()
  return (
    <Sidebar collapsible="icon" label={t('common.menu')}>
      <SidebarHeader className="py-5">
        <Link
          to="/"
          className="flex min-h-9 items-center gap-2 px-1"
          aria-label="Course AI"
          onClick={() => setOpenMobile(false)}
        >
          <CourseAiLogo compact={!isMobile && state === 'collapsed'} />
        </Link>
        {isMobile && (
          <Button
            variant="ghost"
            size="icon"
            className="absolute right-3 top-5"
            aria-label={t('common.close')}
            onClick={() => setOpenMobile(false)}
          >
            <PanelLeftClose />
          </Button>
        )}
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <nav aria-label={t('common.menu')}>
            <SidebarMenu>
              {navItems.map(({ to, label, icon: Icon }) => {
                const active = to === '/' ? pathname === '/' : pathname.startsWith(to)
                return (
                  <SidebarMenuItem key={to}>
                    <SidebarMenuButton asChild isActive={active} tooltip={t(label)} className="h-11">
                      <Link
                        to={to}
                        aria-current={active ? 'page' : undefined}
                        onClick={() => setOpenMobile(false)}
                      >
                        <Icon aria-hidden="true" />
                        <span>{t(label)}</span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                )
              })}
            </SidebarMenu>
          </nav>
        </SidebarGroup>
      </SidebarContent>
      <SidebarSeparator />
      <SidebarFooter className="gap-3 py-4">
        <div className="flex items-center gap-1 group-data-[collapsible=icon]:flex-col">
          <ThemeControl />
          <LanguageControl />
          <span className="ml-auto px-1 group-data-[collapsible=icon]:ml-0">
            {import.meta.env.DEV && import.meta.env.VITE_E2E_MODE === 'true' ? (
              <span
                className="grid size-8 place-items-center rounded-full bg-muted text-xs font-medium"
                aria-label="Test user"
              >
                TU
              </span>
            ) : (
              <UserButton />
            )}
          </span>
        </div>
      </SidebarFooter>
    </Sidebar>
  )
}

export function AppShell() {
  const { t } = useTranslation()
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  const currentPage = navItems.find(({ to }) => to !== '/' && pathname.startsWith(to)) ?? navItems[0]
  return (
    <SidebarProvider defaultOpen={window.matchMedia('(min-width: 1024px)').matches}>
      <a
        href="#main-content"
        className="fixed left-4 top-3 z-[60] -translate-y-24 rounded-md bg-primary px-4 py-3 text-primary-foreground focus:translate-y-0"
      >
        {t('design.skip')}
      </a>
      <WorkspaceNavigation />
      <SidebarInset className="min-w-0">
        <header className="sticky top-0 z-30 flex h-14 shrink-0 items-center gap-3 border-b bg-background/95 px-4 backdrop-blur sm:px-6">
          <SidebarTrigger className="size-9" aria-label={t('common.menu')} />
          <span className="text-sm font-medium">{t(currentPage.label)}</span>
        </header>
        <div
          id="main-content"
          tabIndex={-1}
          className="mx-auto w-full max-w-[1200px] flex-1 px-4 py-7 outline-none sm:px-6 lg:px-8 lg:py-9"
        >
          <Outlet />
        </div>
      </SidebarInset>
    </SidebarProvider>
  )
}
