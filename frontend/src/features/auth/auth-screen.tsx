import { SignIn, SignUp } from '@clerk/react'
import { useTranslation } from 'react-i18next'
import { CourseAiLogo } from '@/shared/ui/course-ai-logo'
import { ThemeControl } from '@/shared/ui/theme-control'

export function AuthScreen({ mode }: { mode: 'sign-in' | 'sign-up' }) {
  const { t } = useTranslation()
  return (
    <main className="flex min-h-svh flex-col bg-background">
      <header className="flex items-center justify-between p-5 sm:px-8">
        <CourseAiLogo />
        <ThemeControl />
      </header>
      <section className="mx-auto flex w-full max-w-[440px] flex-1 flex-col justify-center px-5 py-12">
        <h1 className="mb-3 text-center text-2xl font-semibold tracking-tight">{t('auth.title')}</h1>
        <p className="mb-8 text-center text-sm leading-6 text-muted-foreground">{t('auth.subtitle')}</p>
        {mode === 'sign-in' ? (
          <SignIn routing="path" path="/sign-in" signUpUrl="/sign-up" forceRedirectUrl="/" />
        ) : (
          <SignUp routing="path" path="/sign-up" signInUrl="/sign-in" forceRedirectUrl="/" />
        )}
      </section>
    </main>
  )
}
