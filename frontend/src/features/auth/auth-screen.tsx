import { SignIn, SignUp } from '@clerk/react'
import { BookOpenCheck } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import authStudio from '@/assets/auth-studio.webp'

interface AuthScreenProps {
  mode: 'sign-in' | 'sign-up'
}

export function AuthScreen({ mode }: AuthScreenProps) {
  const { t } = useTranslation()
  const clerkProps = {
    appearance: {
      variables: {
        colorPrimary: '#1f5c46',
        borderRadius: '8px',
        fontFamily: 'Manrope Variable, sans-serif',
      },
      elements: { cardBox: 'shadow-none', card: 'shadow-none border border-border' },
    },
  }

  return (
    <main className="grid min-h-screen bg-background lg:grid-cols-[minmax(0,1.15fr)_minmax(420px,.85fr)]">
      <section className="relative hidden min-h-screen overflow-hidden lg:block">
        <img className="absolute inset-0 size-full object-cover" src={authStudio} alt="" />
        <div className="absolute inset-0 bg-black/20" />
        <div className="absolute inset-x-0 bottom-0 p-12 text-white xl:p-16">
          <div className="mb-8 flex items-center gap-3 text-sm font-bold">
            <BookOpenCheck className="size-6" />
            {t('brand.name')}
          </div>
          <h1 className="max-w-xl text-4xl font-bold leading-tight xl:text-5xl">{t('auth.title')}</h1>
          <p className="mt-5 max-w-lg text-lg leading-8 text-white/85">{t('auth.subtitle')}</p>
        </div>
      </section>
      <section className="flex min-h-screen flex-col">
        <div className="flex items-center gap-2 p-6 text-sm font-bold lg:hidden">
          <BookOpenCheck className="size-5 text-primary" />
          {t('brand.name')}
        </div>
        <div className="flex flex-1 items-center justify-center px-4 py-10">
          {mode === 'sign-in' ? (
            <SignIn
              {...clerkProps}
              routing="path"
              path="/sign-in"
              signUpUrl="/sign-up"
              forceRedirectUrl="/"
            />
          ) : (
            <SignUp
              {...clerkProps}
              routing="path"
              path="/sign-up"
              signInUrl="/sign-in"
              forceRedirectUrl="/"
            />
          )}
        </div>
      </section>
    </main>
  )
}
