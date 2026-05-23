import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Send } from 'lucide-react'

import { Button } from '../../../components/ui/Button'
import { TextField } from '../../../components/ui/TextField'
import { Select } from '../../../components/ui/Select'
import {
  useSetting,
  useTestEmail,
  useUpdateSetting,
  unmask,
} from '../../../hooks/useSettings'

interface EmailConfig {
  enabled: boolean
  provider: 'smtp' | 'resend' | 'elastic_mail'
  from_address: string
  from_name: string
  reply_to: string
  smtp: {
    host: string
    port: number
    username: string
    password: unknown
    starttls: boolean
    use_tls: boolean
  }
  resend: { api_key: unknown }
  elastic_mail: { api_key: unknown; base_url: string }
}

const BLANK: EmailConfig = {
  enabled: false,
  provider: 'smtp',
  from_address: '',
  from_name: '',
  reply_to: '',
  smtp: { host: '', port: 587, username: '', password: '', starttls: true, use_tls: false },
  resend: { api_key: '' },
  elastic_mail: { api_key: '', base_url: 'https://api.elasticemail.com/v4' },
}

export default function EmailSettings() {
  const { t } = useTranslation()
  const q = useSetting<EmailConfig>('email')
  const update = useUpdateSetting<EmailConfig>('email')
  const test = useTestEmail()
  const [form, setForm] = useState<EmailConfig>(BLANK)
  const [savedAt, setSavedAt] = useState<number | null>(null)
  const [testTo, setTestTo] = useState('')

  useEffect(() => {
    if (q.data?.value) {
      // Spread defaults so any newly-introduced fields don't NPE.
      setForm({
        ...BLANK,
        ...q.data.value,
        smtp: { ...BLANK.smtp, ...(q.data.value.smtp ?? {}) },
        resend: { ...BLANK.resend, ...(q.data.value.resend ?? {}) },
        elastic_mail: { ...BLANK.elastic_mail, ...(q.data.value.elastic_mail ?? {}) },
      })
    }
  }, [q.data])

  const onSave = () => {
    update.mutate(unmask(form), { onSuccess: () => setSavedAt(Date.now()) })
  }

  const onTest = () => {
    test.mutate({ to: testTo, config: unmask(form) })
  }

  return (
    <div className="space-y-8">
      <Section
        title={t('admin.settings.email.identity')}
        subtitle={t('admin.settings.email.identity_help')}
      >
        <div className="grid gap-4 sm:grid-cols-2">
          <Toggle
            label={t('admin.settings.email.enabled')}
            checked={form.enabled}
            onChange={(v) => setForm({ ...form, enabled: v })}
            help={t('admin.settings.email.enabled_help')}
          />
          <Select
            label={t('admin.settings.email.provider')}
            value={form.provider}
            onChange={(e) =>
              setForm({ ...form, provider: e.target.value as EmailConfig['provider'] })
            }
            options={[
              { value: 'smtp', label: 'SMTP' },
              { value: 'resend', label: 'Resend' },
              { value: 'elastic_mail', label: 'Elastic Email' },
            ]}
          />
          <TextField
            label={t('admin.settings.email.from_name')}
            value={form.from_name}
            onChange={(e) => setForm({ ...form, from_name: e.target.value })}
          />
          <TextField
            label={t('admin.settings.email.from_address')}
            type="email"
            value={form.from_address}
            onChange={(e) => setForm({ ...form, from_address: e.target.value })}
          />
          <TextField
            label={t('admin.settings.email.reply_to')}
            type="email"
            value={form.reply_to}
            onChange={(e) => setForm({ ...form, reply_to: e.target.value })}
          />
        </div>
      </Section>

      {form.provider === 'smtp' && (
        <Section title="SMTP">
          <div className="grid gap-4 sm:grid-cols-2">
            <TextField
              label={t('admin.settings.email.smtp_host')}
              value={form.smtp.host}
              onChange={(e) => setForm({ ...form, smtp: { ...form.smtp, host: e.target.value } })}
              placeholder="smtp.mailgun.org"
            />
            <TextField
              label={t('admin.settings.email.smtp_port')}
              type="number"
              value={form.smtp.port}
              onChange={(e) =>
                setForm({ ...form, smtp: { ...form.smtp, port: Number(e.target.value) || 0 } })
              }
            />
            <TextField
              label={t('admin.settings.email.smtp_username')}
              value={form.smtp.username}
              onChange={(e) =>
                setForm({ ...form, smtp: { ...form.smtp, username: e.target.value } })
              }
            />
            <SecretField
              label={t('admin.settings.email.smtp_password')}
              value={form.smtp.password}
              onChange={(v) => setForm({ ...form, smtp: { ...form.smtp, password: v } })}
            />
            <Toggle
              label="STARTTLS"
              checked={form.smtp.starttls}
              onChange={(v) => setForm({ ...form, smtp: { ...form.smtp, starttls: v } })}
            />
            <Toggle
              label="TLS (port 465)"
              checked={form.smtp.use_tls}
              onChange={(v) => setForm({ ...form, smtp: { ...form.smtp, use_tls: v } })}
            />
          </div>
        </Section>
      )}

      {form.provider === 'resend' && (
        <Section title="Resend">
          <div className="grid gap-4 sm:grid-cols-2">
            <SecretField
              label={t('admin.settings.email.api_key')}
              value={form.resend.api_key}
              onChange={(v) => setForm({ ...form, resend: { api_key: v } })}
              help={t('admin.settings.email.resend_help')}
            />
          </div>
        </Section>
      )}

      {form.provider === 'elastic_mail' && (
        <Section title="Elastic Email">
          <div className="grid gap-4 sm:grid-cols-2">
            <SecretField
              label={t('admin.settings.email.api_key')}
              value={form.elastic_mail.api_key}
              onChange={(v) =>
                setForm({ ...form, elastic_mail: { ...form.elastic_mail, api_key: v } })
              }
            />
            <TextField
              label={t('admin.settings.email.base_url')}
              value={form.elastic_mail.base_url}
              onChange={(e) =>
                setForm({
                  ...form,
                  elastic_mail: { ...form.elastic_mail, base_url: e.target.value },
                })
              }
            />
          </div>
        </Section>
      )}

      <Section title={t('admin.settings.email.test_title')} subtitle={t('admin.settings.email.test_help')}>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
          <TextField
            label={t('admin.settings.email.test_to')}
            type="email"
            value={testTo}
            onChange={(e) => setTestTo(e.target.value)}
            className="flex-1"
          />
          <Button variant="secondary" onClick={onTest} loading={test.isPending} disabled={!testTo}>
            <Send className="h-4 w-4" aria-hidden />
            {t('admin.settings.email.send_test')}
          </Button>
        </div>
        {test.error && (
          <p className="mt-3 text-sm text-red-600">{(test.error as Error).message}</p>
        )}
        {test.data?.ok && (
          <p className="mt-3 text-sm text-emerald-600">{t('admin.settings.email.test_ok')}</p>
        )}
      </Section>

      <div className="flex items-center gap-3 border-t border-ink-200 pt-4">
        <Button onClick={onSave} loading={update.isPending}>
          {t('common.save')}
        </Button>
        {update.error && (
          <span className="text-sm text-red-600">{(update.error as Error).message}</span>
        )}
        {savedAt && !update.isPending && !update.error && (
          <span className="text-sm text-emerald-600">{t('admin.settings.saved')}</span>
        )}
      </div>
    </div>
  )
}

function Section({
  title,
  subtitle,
  children,
}: {
  title: string
  subtitle?: string
  children: React.ReactNode
}) {
  return (
    <section>
      <h2 className="font-display text-lg font-semibold text-ink-900">{title}</h2>
      {subtitle && <p className="mt-1 text-sm text-ink-600">{subtitle}</p>}
      <div className="mt-4">{children}</div>
    </section>
  )
}

function Toggle({
  label,
  checked,
  onChange,
  help,
}: {
  label: string
  checked: boolean
  onChange: (v: boolean) => void
  help?: string
}) {
  return (
    <label className="flex items-start gap-3 rounded-md border border-ink-200 bg-white p-3 text-sm">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="mt-0.5 h-4 w-4 rounded border-ink-300"
      />
      <div>
        <p className="font-medium text-ink-900">{label}</p>
        {help && <p className="mt-0.5 text-xs text-ink-500">{help}</p>}
      </div>
    </label>
  )
}

/**
 * SecretField renders a password input that:
 *  - shows "•••••• (saved)" when the server returned `{__masked: true}`
 *  - clears that placeholder on focus so the operator can replace it
 *  - keeps the masked sentinel as-is when the operator doesn't type
 */
function SecretField({
  label,
  value,
  onChange,
  help,
}: {
  label: string
  value: unknown
  onChange: (v: string) => void
  help?: string
}) {
  const masked =
    !!value && typeof value === 'object' && (value as Record<string, unknown>).__masked === true
  const stringValue = typeof value === 'string' ? value : ''
  return (
    <TextField
      label={label}
      type="password"
      value={masked ? '' : stringValue}
      placeholder={masked ? '•••••••• (saved)' : ''}
      onChange={(e) => onChange(e.target.value)}
      helperText={help}
    />
  )
}
