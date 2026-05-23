import { useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

import { Button } from '../components/ui/Button'
import { TextField } from '../components/ui/TextField'
import { apiPost, ApiError } from '../lib/api'

/**
 * Submits the (token, new_password) pair. On success, redirects to the
 * login page so the user authenticates with their fresh credentials.
 */
export default function ResetPassword() {
  const { t } = useTranslation()
  const [params] = useSearchParams()
  const nav = useNavigate()
  const token = params.get('token') ?? ''
  const [pwd, setPwd] = useState('')
  const [confirm, setConfirm] = useState('')
  const [pending, setPending] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setErr(null)
    if (pwd.length < 10) {
      setErr(t('reset.too_short'))
      return
    }
    if (pwd !== confirm) {
      setErr(t('register.password_mismatch'))
      return
    }
    setPending(true)
    try {
      await apiPost('/api/v1/auth/password/reset', { token, new_password: pwd })
      nav('/login?reset=ok', { replace: true })
    } catch (e) {
      const msg = e instanceof ApiError && e.message ? e.message : t('reset.error_generic')
      setErr(msg)
    } finally {
      setPending(false)
    }
  }

  if (!token) {
    return (
      <section className="container-page max-w-md py-16">
        <div className="card-soft p-8">
          <h1 className="font-display text-2xl font-bold text-ink-900">{t('reset.invalid_title')}</h1>
          <p className="mt-2 text-sm text-ink-600">{t('reset.invalid_message')}</p>
          <Link to="/forgot-password" className="link-quiet mt-4 inline-block">
            {t('reset.request_new')}
          </Link>
        </div>
      </section>
    )
  }

  return (
    <section className="container-page max-w-md py-16">
      <div className="card-soft p-8">
        <h1 className="font-display text-2xl font-bold text-ink-900">{t('reset.title')}</h1>
        <p className="mt-2 text-sm text-ink-600">{t('reset.subtitle')}</p>
        <form onSubmit={onSubmit} className="mt-6 space-y-4">
          <TextField
            label={t('reset.new_password')}
            type="password"
            autoComplete="new-password"
            required
            value={pwd}
            onChange={(e) => setPwd(e.target.value)}
            helperText={t('register.password_help')}
          />
          <TextField
            label={t('register.confirm_password')}
            type="password"
            autoComplete="new-password"
            required
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
          />
          {err && <p className="text-sm text-red-600">{err}</p>}
          <Button type="submit" loading={pending} fullWidth>
            {t('reset.cta')}
          </Button>
        </form>
      </div>
    </section>
  )
}
