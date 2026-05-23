// Editor for a single owned case. Three sections (basics, persons,
// evidence) are exposed as anchored panels. Submit-for-review lives in
// the footer once the case has at least one person + one attachment.

import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

import { Container } from '../components/ui/Container'
import { Card, CardBody, CardHeader } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { Badge } from '../components/ui/Badge'
import { TextField } from '../components/ui/TextField'
import { TextArea } from '../components/ui/TextArea'
import { Select } from '../components/ui/Select'
import { LocationCascade, type LocationValue } from '../components/forms/LocationCascade'
import { FileDropzone } from '../components/forms/FileDropzone'
import { LoadingState, ErrorState } from '../components/ui/States'

import { useCrimeTypes } from '../hooks/useReferenceData'
import {
  useMyCase,
  usePatchCase,
  useSubmitCase,
  useAddCasePerson,
  useRemoveCasePerson,
} from '../hooks/useCases'
import { useMyPersons, useCreatePerson } from '../hooks/usePersons'
import {
  uploadFileToCase,
  useCaseAttachments,
  type AttachmentKind,
} from '../hooks/useAttachments'

interface UploadState {
  name: string
  pct: number
  status: 'uploading' | 'done' | 'error'
  error?: string
}

export default function MyCaseEdit() {
  const { id } = useParams<{ id: string }>()
  const { t, i18n } = useTranslation()
  const isBN = (i18n.resolvedLanguage ?? 'bn').startsWith('bn')
  const nav = useNavigate()

  const detail = useMyCase(id)
  const patch = usePatchCase(id ?? '')
  const submit = useSubmitCase()
  const addPerson = useAddCasePerson()
  const removePerson = useRemoveCasePerson()
  const myPersons = useMyPersons()
  const createPerson = useCreatePerson()
  const crimes = useCrimeTypes()
  const attachments = useCaseAttachments(id)

  const [titleBN, setTitleBN] = useState('')
  const [titleEN, setTitleEN] = useState('')
  const [summaryBN, setSummaryBN] = useState('')
  const [descBN, setDescBN] = useState('')
  const [tags, setTags] = useState('')
  const [crimeTypeID, setCrimeTypeID] = useState('')
  const [loc, setLoc] = useState<LocationValue>({
    countryId: null,
    divisionId: null,
    districtId: null,
    upazilaId: null,
    text: '',
  })

  const [linkPersonID, setLinkPersonID] = useState('')
  const [linkRole, setLinkRole] = useState<'victim' | 'accused' | 'witness' | 'other'>('victim')
  const [newPersonName, setNewPersonName] = useState('')
  const [newPersonAnonymous, setNewPersonAnonymous] = useState(false)

  const [uploads, setUploads] = useState<UploadState[]>([])
  const [uploadKind, setUploadKind] = useState<AttachmentKind>('public')

  // Hydrate form from server data once detail loads.
  useEffect(() => {
    if (!detail.data) return
    const c = detail.data.case
    setTitleBN(c.title_bn ?? '')
    setTitleEN(c.title_en ?? '')
    setSummaryBN(c.summary_bn ?? '')
    setDescBN(c.description_bn ?? '')
    setTags((c.tags ?? []).join(', '))
    setCrimeTypeID(c.crime_type_id ? String(c.crime_type_id) : '')
    setLoc({
      countryId: c.country_id ?? null,
      divisionId: c.division_id ?? null,
      districtId: c.district_id ?? null,
      upazilaId: c.upazila_id ?? null,
      text: c.location_text ?? '',
    })
  }, [detail.data])

  if (detail.isPending) return <Container width="reading"><LoadingState /></Container>
  if (detail.isError || !detail.data)
    return <Container width="reading"><ErrorState onRetry={() => detail.refetch()} /></Container>

  const c = detail.data.case
  const linkedPersons = detail.data.persons
  const isLocked = c.status !== 'draft' && c.status !== 'rejected'

  const saveDraft = () =>
    patch.mutateAsync({
      title_bn: titleBN,
      title_en: titleEN || null,
      summary_bn: summaryBN || null,
      description_bn: descBN || null,
      tags: tags.split(',').map((t) => t.trim()).filter(Boolean),
      crime_type_id: crimeTypeID ? Number(crimeTypeID) : null,
      country_id: loc.countryId,
      division_id: loc.divisionId,
      district_id: loc.districtId,
      upazila_id: loc.upazilaId,
      location_text: loc.text || null,
    } as Record<string, unknown>)

  const handleAddPerson = async () => {
    if (!linkPersonID || !id) return
    await addPerson.mutateAsync({ caseId: id, personId: linkPersonID, role: linkRole })
    setLinkPersonID('')
  }

  const handleCreateAndLinkPerson = async () => {
    if (!id) return
    const created = await createPerson.mutateAsync({
      primary_type: linkRole,
      full_name_en: newPersonName || null,
      is_anonymous: newPersonAnonymous,
    } as Record<string, unknown>)
    await addPerson.mutateAsync({ caseId: id, personId: created.id, role: linkRole })
    setNewPersonName('')
    setNewPersonAnonymous(false)
  }

  const handleFiles = async (files: File[]) => {
    if (!id) return
    const baseLen = uploads.length
    setUploads((u) => [
      ...u,
      ...files.map<UploadState>((f) => ({ name: f.name, pct: 0, status: 'uploading' })),
    ])
    for (let i = 0; i < files.length; i++) {
      const idx = baseLen + i
      const f = files[i]
      try {
        await uploadFileToCase(id, f, uploadKind, (pct) => {
          setUploads((cur) => {
            const next = cur.slice()
            if (next[idx]) next[idx] = { ...next[idx], pct }
            return next
          })
        })
        setUploads((cur) => {
          const next = cur.slice()
          if (next[idx]) next[idx] = { ...next[idx], pct: 100, status: 'done' }
          return next
        })
        attachments.refetch()
      } catch (err) {
        setUploads((cur) => {
          const next = cur.slice()
          if (next[idx]) next[idx] = { ...next[idx], status: 'error', error: String(err) }
          return next
        })
      }
    }
  }

  const handleSubmit = async () => {
    if (!id) return
    try {
      await submit.mutateAsync(id)
      nav('/me/cases', { replace: true })
    } catch {
      /* shown below */
    }
  }

  return (
    <Container width="reading" className="py-10">
      <nav className="mb-2 text-xs text-ink-500">
        <Link to="/me/cases" className="hover:text-ink-800">
          {t('my_cases.title')}
        </Link>
        <span className="mx-2 text-ink-400">/</span>
        <span className="font-mono">{c.case_number}</span>
      </nav>
      <header className="mb-6 flex items-center justify-between gap-2">
        <div>
          <h1 className="font-display text-2xl font-semibold text-ink-900">
            {c.title_bn || c.case_number}
          </h1>
          <Badge tone="warning">{t(`case_status.${c.status}`)}</Badge>
        </div>
      </header>

      {/* Basics ---------------------------------------------------------- */}
      <Card>
        <CardHeader>
          <h2 className="text-sm font-medium text-ink-800">{t('my_cases.section_basics')}</h2>
        </CardHeader>
        <CardBody className="space-y-4">
          <TextField label={t('my_cases.title_bn')} value={titleBN} onChange={(e) => setTitleBN(e.target.value)} disabled={isLocked} />
          <TextField label={t('my_cases.title_en')} value={titleEN} onChange={(e) => setTitleEN(e.target.value)} disabled={isLocked} />
          <TextArea label={t('my_cases.summary_bn')} rows={3} value={summaryBN} onChange={(e) => setSummaryBN(e.target.value)} disabled={isLocked} />
          <TextArea label={t('my_cases.description_bn')} rows={6} value={descBN} onChange={(e) => setDescBN(e.target.value)} disabled={isLocked} />
          <div className="grid gap-4 sm:grid-cols-2">
            <Select
              label={t('my_cases.crime_type')}
              placeholder={t('my_cases.select_crime_type') ?? ''}
              options={(crimes.data ?? []).map((c) => ({ value: c.id, label: isBN ? c.name_bn : c.name_en }))}
              value={crimeTypeID}
              onChange={(e) => setCrimeTypeID(e.target.value)}
              disabled={isLocked}
            />
            <TextField label={t('my_cases.tags')} value={tags} onChange={(e) => setTags(e.target.value)} disabled={isLocked} />
          </div>
          <LocationCascade value={loc} onChange={setLoc} disabled={isLocked} />
        </CardBody>
        <CardBody className="border-t border-ink-200 text-right">
          <Button onClick={saveDraft} loading={patch.isPending} disabled={isLocked}>
            {t('my_cases.save_draft')}
          </Button>
        </CardBody>
      </Card>

      {/* Persons --------------------------------------------------------- */}
      <Card className="mt-6">
        <CardHeader>
          <h2 className="text-sm font-medium text-ink-800">{t('my_cases.section_persons')}</h2>
          <p className="text-xs text-ink-500">{t('my_cases.persons_help')}</p>
        </CardHeader>
        <CardBody className="space-y-4">
          {linkedPersons.length === 0 && (
            <p className="text-sm text-ink-500">{t('my_cases.no_linked_persons')}</p>
          )}
          {linkedPersons.length > 0 && (
            <ul className="divide-y divide-ink-100">
              {linkedPersons.map((p) => (
                <li key={`${p.person_id}-${p.role}`} className="flex items-center justify-between py-2">
                  <div>
                    <Link to={`/persons/${p.person_slug}`} className="text-sm font-medium text-ink-900 hover:text-brand-700">
                      {p.is_anonymous ? t('persons.anonymous') : (isBN ? p.name_bn : p.name_en) || p.name_bn || p.name_en || '—'}
                    </Link>
                    <p className="text-xs text-ink-500">{t(`person_role.${p.role}`)}</p>
                  </div>
                  {!isLocked && (
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => removePerson.mutate({ caseId: id!, personId: p.person_id, role: p.role })}
                    >
                      {t('common.delete')}
                    </Button>
                  )}
                </li>
              ))}
            </ul>
          )}

          {!isLocked && (
            <div className="grid gap-3 rounded-md bg-ink-50 p-4 sm:grid-cols-3">
              <Select
                label={t('my_cases.role')}
                options={[
                  { value: 'victim', label: t('person_role.victim') },
                  { value: 'accused', label: t('person_role.accused') },
                  { value: 'witness', label: t('person_role.witness') },
                  { value: 'other', label: t('person_role.other') },
                ]}
                value={linkRole}
                onChange={(e) => setLinkRole(e.target.value as typeof linkRole)}
              />
              <Select
                label={t('my_cases.existing_person')}
                placeholder={t('my_cases.select_existing') ?? ''}
                options={(myPersons.data ?? []).map((p) => ({
                  value: p.id,
                  label:
                    (isBN ? p.full_name_bn : p.full_name_en) ||
                    p.full_name_en ||
                    p.full_name_bn ||
                    p.slug,
                }))}
                value={linkPersonID}
                onChange={(e) => setLinkPersonID(e.target.value)}
              />
              <div className="flex items-end">
                <Button fullWidth onClick={handleAddPerson} loading={addPerson.isPending} disabled={!linkPersonID}>
                  {t('my_cases.link_person')}
                </Button>
              </div>
              <div className="sm:col-span-3">
                <p className="text-xs uppercase tracking-wide text-ink-500">
                  {t('my_cases.create_new_person')}
                </p>
                <div className="mt-2 grid gap-3 sm:grid-cols-3">
                  <TextField
                    label={t('my_cases.new_person_name')}
                    value={newPersonName}
                    onChange={(e) => setNewPersonName(e.target.value)}
                  />
                  <label className="flex items-end gap-2 text-sm text-ink-700">
                    <input
                      type="checkbox"
                      checked={newPersonAnonymous}
                      onChange={(e) => setNewPersonAnonymous(e.target.checked)}
                      className="h-4 w-4"
                    />
                    {t('my_cases.anonymous_label')}
                  </label>
                  <div className="flex items-end">
                    <Button
                      fullWidth
                      variant="secondary"
                      onClick={handleCreateAndLinkPerson}
                      loading={createPerson.isPending}
                      disabled={!newPersonName && !newPersonAnonymous}
                    >
                      {t('my_cases.create_and_link')}
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          )}
        </CardBody>
      </Card>

      {/* Evidence -------------------------------------------------------- */}
      <Card className="mt-6">
        <CardHeader>
          <h2 className="text-sm font-medium text-ink-800">{t('my_cases.section_evidence')}</h2>
          <p className="text-xs text-ink-500">{t('my_cases.evidence_help')}</p>
        </CardHeader>
        <CardBody className="space-y-4">
          {!isLocked && (
            <div className="flex items-center gap-3">
              <Select
                label={t('my_cases.attachment_kind')}
                options={[
                  { value: 'public', label: t('attachment_kind.public') },
                  { value: 'hidden', label: t('attachment_kind.hidden') },
                ]}
                value={uploadKind}
                onChange={(e) => setUploadKind(e.target.value as AttachmentKind)}
              />
              <div className="flex-1">
                <FileDropzone onFiles={handleFiles} hint={t('my_cases.dropzone_hint')} />
              </div>
            </div>
          )}
          {uploads.length > 0 && (
            <ul className="space-y-2">
              {uploads.map((u, idx) => (
                <li key={`${u.name}-${idx}`} className="flex items-center justify-between text-xs">
                  <span className="truncate text-ink-700">{u.name}</span>
                  <span className={u.status === 'error' ? 'text-red-600' : 'text-ink-500'}>
                    {u.status === 'done'
                      ? '✓'
                      : u.status === 'error'
                      ? u.error
                      : `${Math.round(u.pct)}%`}
                  </span>
                </li>
              ))}
            </ul>
          )}

          {attachments.data && attachments.data.length > 0 && (
            <div>
              <p className="text-xs uppercase tracking-wide text-ink-500">
                {t('my_cases.uploaded_files')}
              </p>
              <ul className="mt-2 divide-y divide-ink-100 rounded-md border border-ink-200 bg-white">
                {attachments.data.map((a) => (
                  <li key={a.id} className="flex items-center justify-between p-3">
                    <div className="min-w-0">
                      <p className="truncate font-mono text-xs text-ink-700">{a.stored_filename}</p>
                      <p className="text-xs text-ink-500">
                        {a.mime_type} · {formatSize(a.size_bytes)} · {t(`attachment_kind.${a.kind}`)}
                      </p>
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </CardBody>
      </Card>

      {/* Submit ---------------------------------------------------------- */}
      {!isLocked && (
        <div className="mt-6 flex justify-end gap-3">
          <Button variant="secondary" onClick={() => nav('/me/cases')}>
            {t('common.back')}
          </Button>
          <Button onClick={handleSubmit} loading={submit.isPending}>
            {t('my_cases.submit_for_review')}
          </Button>
        </div>
      )}
    </Container>
  )
}

function formatSize(n: number) {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}
