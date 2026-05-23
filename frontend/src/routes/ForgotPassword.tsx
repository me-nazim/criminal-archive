import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

import { Button } from '../components/ui/Button'
import { TextField } from '../components/ui/TextField'
import { apiPost, ApiError } from '../lib/api'

/**
 * Request a password-reset email. Always succeeds visually so attackers
 * can't enumerate registered emails — the backend silently no-ops when
 * the email is unknown.
 */
export default function ForgotPassword() {
  const { t } = useTranslation()
  const [email, setEmail] = useState('')
  const [done, setDone] = useState(false)
  const [pending, setPending] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setPending(true)
    setErr(null)
    try {
      await apiPost('/api/v1/auth/password/forgot', { email })
      setDone(true)
    } catch (e) {
      const message =
        e instanceof ApiError && e.message ? e.message : t('forgot.error_generic')
      setErr(message)
    } finally {
      setPending(false)
    }
  }

  return (
    <section className="container-page max-w-md py-16">
      <div className="card-soft p-8">
        <h1 className="font-display text-2xl font-bold text-ink-900">
          {t('forgot.title')}
        </h1>
        <p className="mt-2 text-sm text-ink-600">{t('forgot.subtitle')}</p>
        {done ? (
          <div className="mt-6 rounded-md border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-800">
            {t('forgot.sent_message')}
          </div>
        ) : (
          <form onSubmit={onSubmit} className="mt-6 space-y-4">
            <TextField
              label={t('common.email')}
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
            {err && <p className="text-sm text-red-600">{err}</p>}
            <Button type="submit" loading={pending} fullWidth>
              {t('forgot.cta')}
            </Button>
          </form>
        )}
        <p className="mt-6 text-center text-sm text-ink-600">
          <Link to="/login" className="link-quiet">
            {t('forgot.back_to_login')}
          </Link>
        </p>
      </div>
    </section>
  )
}
