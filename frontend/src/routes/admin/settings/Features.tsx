import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '../../../components/ui/Button'
import { Select } from '../../../components/ui/Select'
import { TextArea } from '../../../components/ui/TextArea'
import { useSetting, useUpdateSetting } from '../../../hooks/useSettings'

interface Features {
  allow_public_registration: boolean
  require_email_verification: boolean
  maintenance_mode: boolean
  maintenance_message_bn: string
  maintenance_message_en: string
  banner_enabled: boolean
  banner_level: 'info' | 'warning' | 'critical'
  banner_message_bn: string
  banner_message_en: string
}

const BLANK: Features = {
  allow_public_registration: true,
  require_email_verification: false,
  maintenance_mode: false,
  maintenance_message_bn: '',
  maintenance_message_en: '',
  banner_enabled: false,
  banner_level: 'info',
  banner_message_bn: '',
  banner_message_en: '',
}

export default function FeaturesSettings() {
  const { t } = useTranslation()
  const q = useSetting<Features>('features')
  const update = useUpdateSetting<Features>('features')
  const [form, setForm] = useState<Features>(BLANK)
  const [savedAt, setSavedAt] = useState<number | null>(null)

  useEffect(() => {
    if (q.data?.value) setForm({ ...BLANK, ...q.data.value })
  }, [q.data])

  const onSave = () => {
    update.mutate(form, { onSuccess: () => setSavedAt(Date.now()) })
  }

  return (
    <div className="space-y-8">
      <Section title={t('admin.settings.features.access')}>
        <div className="grid gap-3 sm:grid-cols-2">
          <Toggle
            label={t('admin.settings.features.allow_public_registration')}
            checked={form.allow_public_registration}
            onChange={(v) => setForm({ ...form, allow_public_registration: v })}
          />
          <Toggle
            label={t('admin.settings.features.require_email_verification')}
            checked={form.require_email_verification}
            onChange={(v) => setForm({ ...form, require_email_verification: v })}
          />
        </div>
      </Section>

      <Section
        title={t('admin.settings.features.banner')}
        subtitle={t('admin.settings.features.banner_help')}
      >
        <div className="grid gap-3 sm:grid-cols-2">
          <Toggle
            label={t('admin.settings.features.banner_enabled')}
            checked={form.banner_enabled}
            onChange={(v) => setForm({ ...form, banner_enabled: v })}
          />
          <Select
            label={t('admin.settings.features.banner_level')}
            value={form.banner_level}
            onChange={(e) => setForm({ ...form, banner_level: e.target.value as Features['banner_level'] })}
            options={[
              { value: 'info', label: 'Info' },
              { value: 'warning', label: 'Warning' },
              { value: 'critical', label: 'Critical' },
            ]}
          />
        </div>
        <div className="mt-3 grid gap-3 sm:grid-cols-2">
          <TextArea
            label={t('admin.settings.features.banner_message_bn')}
            value={form.banner_message_bn}
            onChange={(e) => setForm({ ...form, banner_message_bn: e.target.value })}
            rows={2}
          />
          <TextArea
            label={t('admin.settings.features.banner_message_en')}
            value={form.banner_message_en}
            onChange={(e) => setForm({ ...form, banner_message_en: e.target.value })}
            rows={2}
          />
        </div>
      </Section>

      <Section
        title={t('admin.settings.features.maintenance')}
        subtitle={t('admin.settings.features.maintenance_help')}
      >
        <Toggle
          label={t('admin.settings.features.maintenance_mode')}
          checked={form.maintenance_mode}
          onChange={(v) => setForm({ ...form, maintenance_mode: v })}
        />
        <div className="mt-3 grid gap-3 sm:grid-cols-2">
          <TextArea
            label={t('admin.settings.features.maintenance_message_bn')}
            value={form.maintenance_message_bn}
            onChange={(e) => setForm({ ...form, maintenance_message_bn: e.target.value })}
            rows={2}
          />
          <TextArea
            label={t('admin.settings.features.maintenance_message_en')}
            value={form.maintenance_message_en}
            onChange={(e) => setForm({ ...form, maintenance_message_en: e.target.value })}
            rows={2}
          />
        </div>
      </Section>

      <div className="flex items-center gap-3 border-t border-ink-200 pt-4">
        <Button onClick={onSave} loading={update.isPending}>
          {t('common.save')}
        </Button>
        {update.error && <span className="text-sm text-red-600">{(update.error as Error).message}</span>}
        {savedAt && !update.isPending && !update.error && (
          <span className="text-sm text-emerald-600">{t('admin.settings.saved')}</span>
        )}
      </div>
    </div>
  )
}

function Section({ title, subtitle, children }: { title: string; subtitle?: string; children: React.ReactNode }) {
  return (
    <section>
      <h2 className="font-display text-lg font-semibold text-ink-900">{title}</h2>
      {subtitle && <p className="mt-1 text-sm text-ink-600">{subtitle}</p>}
      <div className="mt-4">{children}</div>
    </section>
  )
}

function Toggle({ label, checked, onChange }: { label: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <label className="flex items-start gap-3 rounded-md border border-ink-200 bg-white p-3 text-sm">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="mt-0.5 h-4 w-4 rounded border-ink-300"
      />
      <span className="font-medium text-ink-900">{label}</span>
    </label>
  )
}
