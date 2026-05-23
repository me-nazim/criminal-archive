import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CheckCircle2, PlugZap } from 'lucide-react'

import { Button } from '../../../components/ui/Button'
import { Select } from '../../../components/ui/Select'
import { TextField } from '../../../components/ui/TextField'
import {
  unmask,
  useSetting,
  useTestStorage,
  useUpdateSetting,
} from '../../../hooks/useSettings'

interface StorageConfig {
  enabled: boolean
  driver: 'r2' | 'aws_s3' | 'minio' | 's3_compatible'
  bucket: string
  region: string
  endpoint: string
  access_key: string
  secret_key: unknown
  public_base_url: string
  force_path_style: boolean
}

const BLANK: StorageConfig = {
  enabled: false,
  driver: 's3_compatible',
  bucket: '',
  region: 'auto',
  endpoint: '',
  access_key: '',
  secret_key: '',
  public_base_url: '',
  force_path_style: false,
}

const DRIVER_HINTS: Record<StorageConfig['driver'], string> = {
  r2: 'Cloudflare R2 — endpoint like https://<account>.r2.cloudflarestorage.com',
  aws_s3: 'AWS S3 — leave endpoint empty to use the official region resolver',
  minio: 'MinIO — endpoint typically http://minio:9000 (path style on)',
  s3_compatible: 'Generic S3-compatible (Backblaze B2, Wasabi, Linode, …)',
}

export default function StorageSettings() {
  const { t } = useTranslation()
  const q = useSetting<StorageConfig>('storage')
  const update = useUpdateSetting<StorageConfig>('storage')
  const test = useTestStorage()
  const [form, setForm] = useState<StorageConfig>(BLANK)
  const [savedAt, setSavedAt] = useState<number | null>(null)

  useEffect(() => {
    if (q.data?.value) {
      setForm({ ...BLANK, ...q.data.value })
    }
  }, [q.data])

  const onSave = () => {
    update.mutate(unmask(form), { onSuccess: () => setSavedAt(Date.now()) })
  }

  const onTest = () => {
    test.mutate(unmask(form))
  }

  return (
    <div className="space-y-8">
      <Section
        title={t('admin.settings.storage.driver_section')}
        subtitle={t('admin.settings.storage.driver_help')}
      >
        <div className="grid gap-4 sm:grid-cols-2">
          <Toggle
            label={t('admin.settings.storage.enabled')}
            checked={form.enabled}
            onChange={(v) => setForm({ ...form, enabled: v })}
          />
          <Select
            label={t('admin.settings.storage.driver')}
            value={form.driver}
            onChange={(e) => {
              const d = e.target.value as StorageConfig['driver']
              setForm((f) => ({
                ...f,
                driver: d,
                force_path_style: d === 'minio' ? true : f.force_path_style,
                region: f.region || (d === 'aws_s3' ? 'us-east-1' : 'auto'),
              }))
            }}
            options={[
              { value: 'r2', label: 'Cloudflare R2' },
              { value: 'aws_s3', label: 'AWS S3' },
              { value: 'minio', label: 'MinIO' },
              { value: 's3_compatible', label: 'S3-compatible (Backblaze B2 etc.)' },
            ]}
            helperText={DRIVER_HINTS[form.driver]}
          />
        </div>
      </Section>

      <Section title={t('admin.settings.storage.connection')}>
        <div className="grid gap-4 sm:grid-cols-2">
          <TextField
            label={t('admin.settings.storage.bucket')}
            value={form.bucket}
            onChange={(e) => setForm({ ...form, bucket: e.target.value })}
          />
          <TextField
            label={t('admin.settings.storage.region')}
            value={form.region}
            onChange={(e) => setForm({ ...form, region: e.target.value })}
          />
          <TextField
            label={t('admin.settings.storage.endpoint')}
            value={form.endpoint}
            onChange={(e) => setForm({ ...form, endpoint: e.target.value })}
            placeholder="https://..."
          />
          <TextField
            label={t('admin.settings.storage.public_base_url')}
            value={form.public_base_url}
            onChange={(e) => setForm({ ...form, public_base_url: e.target.value })}
            placeholder="https://cdn.example.com/bucket"
            helperText={t('admin.settings.storage.public_base_url_help')}
          />
          <TextField
            label={t('admin.settings.storage.access_key')}
            value={form.access_key}
            onChange={(e) => setForm({ ...form, access_key: e.target.value })}
          />
          <SecretField
            label={t('admin.settings.storage.secret_key')}
            value={form.secret_key}
            onChange={(v) => setForm({ ...form, secret_key: v })}
          />
          <Toggle
            label={t('admin.settings.storage.force_path_style')}
            checked={form.force_path_style}
            onChange={(v) => setForm({ ...form, force_path_style: v })}
            help={t('admin.settings.storage.force_path_style_help')}
          />
        </div>
      </Section>

      <Section title={t('admin.settings.storage.test_title')} subtitle={t('admin.settings.storage.test_help')}>
        <div className="flex items-center gap-3">
          <Button variant="secondary" onClick={onTest} loading={test.isPending}>
            <PlugZap className="h-4 w-4" aria-hidden />
            {t('admin.settings.storage.test_button')}
          </Button>
          {test.data?.ok && (
            <span className="inline-flex items-center gap-1 text-sm text-emerald-600">
              <CheckCircle2 className="h-4 w-4" aria-hidden />
              {t('admin.settings.storage.test_ok')}
            </span>
          )}
          {test.error && (
            <span className="text-sm text-red-600">{(test.error as Error).message}</span>
          )}
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

function Section({ title, subtitle, children }: { title: string; subtitle?: string; children: React.ReactNode }) {
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

function SecretField({
  label,
  value,
  onChange,
}: {
  label: string
  value: unknown
  onChange: (v: string) => void
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
    />
  )
}
