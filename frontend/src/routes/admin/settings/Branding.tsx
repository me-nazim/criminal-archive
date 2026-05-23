import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '../../../components/ui/Button'
import { TextField } from '../../../components/ui/TextField'
import { useSetting, useUpdateSetting } from '../../../hooks/useSettings'
import type { Branding } from '../../../lib/branding'

const BLANK: Branding = {
  site_name_bn: '',
  site_name_en: '',
  short_name: '',
  tagline_bn: '',
  tagline_en: '',
  primary_color: '#e8501f',
  accent_color: '#0f1320',
  logo_url: '',
  favicon_url: '',
  support_email: '',
  social: { twitter: '', facebook: '', youtube: '', github: '' },
}

export default function BrandingSettings() {
  const { t } = useTranslation()
  const q = useSetting<Branding>('branding')
  const update = useUpdateSetting<Branding>('branding')
  const [form, setForm] = useState<Branding>(BLANK)
  const [savedAt, setSavedAt] = useState<number | null>(null)

  useEffect(() => {
    if (q.data?.value) setForm({ ...BLANK, ...q.data.value, social: { ...(BLANK.social ?? {}), ...(q.data.value.social ?? {}) } })
  }, [q.data])

  const onSave = () => {
    update.mutate(form, {
      onSuccess: () => setSavedAt(Date.now()),
    })
  }

  const setField = <K extends keyof Branding>(k: K, v: Branding[K]) =>
    setForm((f) => ({ ...f, [k]: v }))

  return (
    <div className="space-y-8">
      <Section title={t('admin.settings.branding.identity')} subtitle={t('admin.settings.branding.identity_help')}>
        <div className="grid gap-4 sm:grid-cols-2">
          <TextField
            label={t('admin.settings.branding.site_name_bn')}
            value={form.site_name_bn}
            onChange={(e) => setField('site_name_bn', e.target.value)}
          />
          <TextField
            label={t('admin.settings.branding.site_name_en')}
            value={form.site_name_en}
            onChange={(e) => setField('site_name_en', e.target.value)}
          />
          <TextField
            label={t('admin.settings.branding.short_name')}
            value={form.short_name}
            onChange={(e) => setField('short_name', e.target.value)}
          />
          <TextField
            label={t('admin.settings.branding.support_email')}
            type="email"
            value={form.support_email ?? ''}
            onChange={(e) => setField('support_email', e.target.value)}
          />
          <TextField
            label={t('admin.settings.branding.tagline_bn')}
            value={form.tagline_bn}
            onChange={(e) => setField('tagline_bn', e.target.value)}
          />
          <TextField
            label={t('admin.settings.branding.tagline_en')}
            value={form.tagline_en}
            onChange={(e) => setField('tagline_en', e.target.value)}
          />
        </div>
      </Section>

      <Section title={t('admin.settings.branding.visuals')} subtitle={t('admin.settings.branding.visuals_help')}>
        <div className="grid gap-4 sm:grid-cols-2">
          <ColorField
            label={t('admin.settings.branding.primary_color')}
            value={form.primary_color}
            onChange={(v) => setField('primary_color', v)}
          />
          <ColorField
            label={t('admin.settings.branding.accent_color')}
            value={form.accent_color}
            onChange={(v) => setField('accent_color', v)}
          />
          <TextField
            label={t('admin.settings.branding.logo_url')}
            placeholder="https://..."
            value={form.logo_url ?? ''}
            onChange={(e) => setField('logo_url', e.target.value)}
            helperText={t('admin.settings.branding.logo_help')}
          />
          <TextField
            label={t('admin.settings.branding.favicon_url')}
            placeholder="https://..."
            value={form.favicon_url ?? ''}
            onChange={(e) => setField('favicon_url', e.target.value)}
          />
        </div>
      </Section>

      <Section title={t('admin.settings.branding.social')}>
        <div className="grid gap-4 sm:grid-cols-2">
          {(['twitter', 'facebook', 'youtube', 'github'] as const).map((k) => (
            <TextField
              key={k}
              label={k}
              placeholder="https://..."
              value={form.social?.[k] ?? ''}
              onChange={(e) =>
                setField('social', { ...(form.social ?? {}), [k]: e.target.value })
              }
            />
          ))}
        </div>
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

function ColorField({
  label,
  value,
  onChange,
}: {
  label: string
  value: string
  onChange: (v: string) => void
}) {
  return (
    <label className="block">
      <span className="mb-1 block text-sm font-medium text-ink-800">{label}</span>
      <div className="flex items-center gap-2">
        <input
          type="color"
          value={value || '#000000'}
          onChange={(e) => onChange(e.target.value)}
          className="h-10 w-12 cursor-pointer rounded-md border border-ink-300 p-1"
        />
        <input
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="h-10 flex-1 rounded-md border border-ink-300 px-3 text-sm font-mono"
        />
      </div>
    </label>
  )
}
