import { useEffect } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'

import { Container } from '../components/ui/Container'
import { Card, CardBody, CardFooter, CardHeader } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { TextField } from '../components/ui/TextField'
import { useLoginMutation } from '../hooks/useAuth'
import { useAuthStore } from '../lib/auth-store'
import { ApiError } from '../lib/api'

const schema = z.object({
  email: z.string().email(),
  password: z.string().min(1),
})
type FormValues = z.infer<typeof schema>

export default function Login() {
  const { t } = useTranslation()
  const nav = useNavigate()
  const location = useLocation()
  const user = useAuthStore((s) => s.user)
  const login = useLoginMutation()

  const from = (location.state as { from?: string } | null)?.from ?? '/me'

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({ resolver: zodResolver(schema) })

  useEffect(() => {
    if (user) nav(from, { replace: true })
  }, [user, nav, from])

  const onSubmit = handleSubmit(async (values) => {
    try {
      await login.mutateAsync(values)
      nav(from, { replace: true })
    } catch {
      // Errors are handled below via login.error.
    }
  })

  const errorMessage = login.error
    ? mapAuthError(login.error, t('login.error_generic'))
    : null

  return (
    <Container width="narrow" className="py-16">
      <Card>
        <CardHeader>
          <h1 className="font-display text-xl font-semibold text-ink-900">{t('login.title')}</h1>
          <p className="mt-1 text-sm text-ink-600">{t('login.subtitle')}</p>
        </CardHeader>
        <form onSubmit={onSubmit} noValidate>
          <CardBody className="space-y-4">
            {errorMessage && (
              <div role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
                {errorMessage}
              </div>
            )}
            <TextField
              label={t('common.email')}
              type="email"
              autoComplete="email"
              showRequired
              errorText={errors.email?.message}
              {...register('email')}
            />
            <TextField
              label={t('common.password')}
              type="password"
              autoComplete="current-password"
              showRequired
              errorText={errors.password?.message}
              {...register('password')}
            />
            <Button type="submit" loading={login.isPending} fullWidth>
              {t('login.cta')}
            </Button>
          </CardBody>
        </form>
        <CardFooter className="flex flex-col gap-2 text-sm sm:flex-row sm:items-center sm:justify-between">
          <span className="text-ink-600">{t('login.no_account')}</span>
          <div className="flex items-center gap-4">
            <Link to="/forgot-password" className="link-quiet">
              {t('login.forgot_password')}
            </Link>
            <Link to="/register" className="font-medium text-brand-600 hover:underline">
              {t('login.register_link')}
            </Link>
          </div>
        </CardFooter>
      </Card>
    </Container>
  )
}

function mapAuthError(err: Error, fallback: string): string {
  if (err instanceof ApiError) {
    if (err.code === 'unauthenticated') return 'Invalid email or password.'
    if (err.code === 'account_pending')
      return 'Your account is awaiting admin approval.'
    if (err.code === 'account_suspended') return 'Your account is suspended.'
    if (err.code === 'account_rejected') return 'Your account has been rejected.'
    return err.message || fallback
  }
  return fallback
}
