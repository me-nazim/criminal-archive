import { Link, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'

import { Container } from '../components/ui/Container'
import { Card, CardBody, CardFooter, CardHeader } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { TextField } from '../components/ui/TextField'
import { useRegisterMutation } from '../hooks/useAuth'
import { ApiError } from '../lib/api'

const schema = z
  .object({
    full_name: z.string().min(1, 'required'),
    email: z.string().email(),
    password: z.string().min(10, 'min10'),
    confirm_password: z.string(),
    phone: z.string().optional(),
  })
  .refine((d) => d.password === d.confirm_password, {
    path: ['confirm_password'],
    message: 'mismatch',
  })

type FormValues = z.infer<typeof schema>

export default function Register() {
  const { t } = useTranslation()
  const nav = useNavigate()
  const reg = useRegisterMutation()

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({ resolver: zodResolver(schema) })

  const onSubmit = handleSubmit(async (values) => {
    try {
      await reg.mutateAsync({
        email: values.email,
        password: values.password,
        full_name: values.full_name,
        phone: values.phone || null,
      })
      nav('/register/pending', { replace: true })
    } catch {
      /* surfaced below */
    }
  })

  const errorMessage = reg.error ? mapErr(reg.error, t('register.error_generic')) : null

  return (
    <Container width="narrow" className="py-16">
      <Card>
        <CardHeader>
          <h1 className="font-display text-xl font-semibold text-ink-900">
            {t('register.title')}
          </h1>
          <p className="mt-1 text-sm text-ink-600">{t('register.subtitle')}</p>
        </CardHeader>
        <form onSubmit={onSubmit} noValidate>
          <CardBody className="space-y-4">
            {errorMessage && (
              <div role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
                {errorMessage}
              </div>
            )}
            <TextField
              label={t('common.full_name')}
              showRequired
              autoComplete="name"
              errorText={errors.full_name && t('common.required_field')}
              {...register('full_name')}
            />
            <TextField
              label={t('common.email')}
              type="email"
              showRequired
              autoComplete="email"
              errorText={errors.email && t('common.invalid_email')}
              {...register('email')}
            />
            <TextField
              label={t('common.phone')}
              type="tel"
              autoComplete="tel"
              {...register('phone')}
            />
            <TextField
              label={t('common.password')}
              type="password"
              autoComplete="new-password"
              showRequired
              helperText={t('register.password_help')}
              errorText={errors.password && t('register.password_help')}
              {...register('password')}
            />
            <TextField
              label={t('register.confirm_password')}
              type="password"
              autoComplete="new-password"
              showRequired
              errorText={errors.confirm_password && t('register.password_mismatch')}
              {...register('confirm_password')}
            />
            <Button type="submit" loading={reg.isPending} fullWidth>
              {t('register.cta')}
            </Button>
          </CardBody>
        </form>
        <CardFooter className="flex items-center justify-between text-sm">
          <span className="text-ink-600">{t('register.have_account')}</span>
          <Link to="/login" className="font-medium text-brand-600 hover:underline">
            {t('login.cta')}
          </Link>
        </CardFooter>
      </Card>
    </Container>
  )
}

function mapErr(err: Error, fallback: string): string {
  if (err instanceof ApiError) {
    if (err.code === 'email_taken') return 'An account with that email already exists.'
    if (err.code === 'validation_error') return err.message || fallback
    return err.message || fallback
  }
  return fallback
}
