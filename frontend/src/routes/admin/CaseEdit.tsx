// Admin case editor: same fields as the user-side editor plus internal
// notes, hidden/internal attachments, verification controls and
// publish / unpublish actions.

import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Eye, Trash2 } from 'lucide-react'

import { Card, CardBody, CardHeader } from '../../components/ui/Card'
import { Button } from '../../components/ui/Button'
import { Badge } from '../../components/ui/Badge'
import { TextField } from '../../components/ui/TextField'
import { TextArea } from '../../components/ui/TextArea'
import { Select } from '../../components/ui/Select'
import { LoadingState, ErrorState } from '../../components/ui/States'
import {
  useAdminCase,
  usePatchCase,
  usePublishCase,
  useUnpublishCase,
  useDeleteCase,
  useVerifyCase,
} from '../../hooks/useCases'
import {
  useCaseAttachments,
  useDeleteAttachment,
  useRequestAttachmentDownload,
  useUpdateAttachment,
} from '../../hooks/useAttachments'

export default function AdminCaseEdit() {
  const { id } = useParams<{ id: string }>()
  const { t, i18n } = useTranslation()
  const isBN = (i18n.resolvedLanguage ?? 'bn').startsWith('bn')
  const nav = useNavigate()

  const detail = useAdminCase(id)
  const patch = usePatchCase(id ?? '')
  const verify = useVerifyCase()
  const publish = usePublishCase()
  const unpublish = useUnpublishCase()
  const del = useDeleteCase()
  const attachments = useCaseAttachments(id)
  const deleteAttach = useDeleteAttachment()
  const requestDownload = useRequestAttachmentDownload()
  const updateAttach = useUpdateAttachment()

  const [internalNotes, setInternalNotes] = useState('')
  const [titleBN, setTitleBN] = useState('')
  const [descBN, setDescBN] = useState('')
  const [verifyReason, setVerifyReason] = useState('')

  useEffect(() => {
    if (!detail.data) return
    setInternalNotes(detail.data.case.internal_notes ?? '')
    setTitleBN(detail.data.case.title_bn ?? '')
    setDescBN(detail.data.case.description_bn ?? '')
  }, [detail.data])

  if (detail.isPending) return <LoadingState />
  if (detail.isError || !detail.data)
    return <ErrorState onRetry={() => detail.refetch()} />

  const c = detail.data.case
  const persons = detail.data.persons

  return (
    <div className="space-y-6">
      <header>
        <p className="text-xs text-ink-500">
          <Link to="/admin/cases" className="hover:text-ink-800">
            {t('admin.cases.title')}
          </Link>
        </p>
        <div className="mt-1 flex flex-wrap items-center gap-2">
          <h1 className="font-display text-2xl font-semibold text-ink-900">
            {c.title_bn || c.case_number}
          </h1>
          <Badge tone="warning">{t(`case_status.${c.status}`)}</Badge>
          <span className="font-mono text-xs text-ink-500">{c.case_number}</span>
        </div>
      </header>

      {/* Internal notes ---------------------------------------------------- */}
      <Card>
        <CardHeader>
          <h2 className="text-sm font-medium text-ink-800">{t('admin.case.internal_notes_title')}</h2>
          <p className="text-xs text-ink-500">{t('admin.case.internal_notes_help')}</p>
        </CardHeader>
        <CardBody className="space-y-3">
          <TextArea rows={5} value={internalNotes} onChange={(e) => setInternalNotes(e.target.value)} />
          <div className="flex justify-end">
            <Button
              size="sm"
              loading={patch.isPending}
              onClick={() => patch.mutate({ internal_notes: internalNotes } as Record<string, unknown>)}
            >
              {t('common.save')}
            </Button>
          </div>
        </CardBody>
      </Card>

      {/* Public fields (admin can override) -------------------------------- */}
      <Card>
        <CardHeader>
          <h2 className="text-sm font-medium text-ink-800">{t('admin.case.public_fields')}</h2>
        </CardHeader>
        <CardBody className="space-y-3">
          <TextField label={t('my_cases.title_bn')} value={titleBN} onChange={(e) => setTitleBN(e.target.value)} />
          <TextArea label={t('my_cases.description_bn')} rows={6} value={descBN} onChange={(e) => setDescBN(e.target.value)} />
          <div className="flex justify-end">
            <Button
              size="sm"
              loading={patch.isPending}
              onClick={() =>
                patch.mutate({
                  title_bn: titleBN,
                  description_bn: descBN,
                } as Record<string, unknown>)
              }
            >
              {t('common.save')}
            </Button>
          </div>
        </CardBody>
      </Card>

      {/* Persons readonly summary ------------------------------------------ */}
      <Card>
        <CardHeader>
          <h2 className="text-sm font-medium text-ink-800">{t('cases.persons_title')}</h2>
        </CardHeader>
        <CardBody>
          {persons.length === 0 ? (
            <p className="text-sm text-ink-500">{t('my_cases.no_linked_persons')}</p>
          ) : (
            <ul className="divide-y divide-ink-100">
              {persons.map((p) => (
                <li key={`${p.person_id}-${p.role}`} className="flex items-center justify-between py-2">
                  <div>
                    <Link to={`/persons/${p.person_slug}`} className="text-sm font-medium text-ink-900 hover:text-brand-700">
                      {p.is_anonymous ? t('persons.anonymous') : (isBN ? p.name_bn : p.name_en) || p.name_bn || p.name_en}
                    </Link>
                    <p className="text-xs text-ink-500">{t(`person_role.${p.role}`)}</p>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </CardBody>
      </Card>

      {/* Attachments (full visibility) ------------------------------------- */}
      <Card>
        <CardHeader>
          <h2 className="text-sm font-medium text-ink-800">{t('admin.case.attachments_title')}</h2>
        </CardHeader>
        <CardBody>
          {attachments.isPending && <LoadingState />}
          {attachments.data && attachments.data.length === 0 && (
            <p className="text-sm text-ink-500">{t('admin.case.no_attachments')}</p>
          )}
          {attachments.data && attachments.data.length > 0 && (
            <ul className="divide-y divide-ink-100 rounded-md border border-ink-200 bg-white">
              {attachments.data.map((a) => (
                <li key={a.id} className="flex items-center justify-between gap-3 p-3 text-sm">
                  <div className="min-w-0">
                    <p className="truncate font-mono text-xs text-ink-800">{a.stored_filename}</p>
                    <p className="text-xs text-ink-500">
                      <Badge tone={a.kind === 'public' ? 'success' : a.kind === 'hidden' ? 'warning' : 'danger'}>
                        {t(`attachment_kind.${a.kind}`)}
                      </Badge>{' '}
                      {a.mime_type} · {formatSize(a.size_bytes)}
                    </p>
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    <Select
                      aria-label="kind"
                      className="h-8 px-2 py-1 text-xs"
                      options={[
                        { value: 'public', label: t('attachment_kind.public') },
                        { value: 'hidden', label: t('attachment_kind.hidden') },
                        { value: 'internal', label: t('attachment_kind.internal') },
                      ]}
                      value={a.kind}
                      onChange={(e) =>
                        updateAttach.mutate({
                          caseId: c.id,
                          attachmentId: a.id,
                          payload: { kind: e.target.value as 'public' | 'hidden' | 'internal' },
                        })
                      }
                    />
                    <Button
                      size="sm"
                      variant="ghost"
                      leftIcon={<Eye className="h-4 w-4" aria-hidden />}
                      onClick={async () => {
                        const res = await requestDownload.mutateAsync(a.id)
                        window.open(res.url, '_blank', 'noopener,noreferrer')
                      }}
                    >
                      {t('admin.case.download')}
                    </Button>
                    <Button
                      size="sm"
                      variant="danger"
                      leftIcon={<Trash2 className="h-4 w-4" aria-hidden />}
                      onClick={() => deleteAttach.mutate({ caseId: c.id, attachmentId: a.id })}
                      loading={deleteAttach.isPending && deleteAttach.variables?.attachmentId === a.id}
                    />
                  </div>
                </li>
              ))}
            </ul>
          )}
        </CardBody>
      </Card>

      {/* Workflow actions ----------------------------------------------- */}
      <Card>
        <CardHeader>
          <h2 className="text-sm font-medium text-ink-800">{t('admin.case.workflow_title')}</h2>
        </CardHeader>
        <CardBody className="space-y-3">
          {c.status === 'in_verification' && (
            <div className="space-y-2">
              <TextField
                label={t('admin.case.verify_reason')}
                value={verifyReason}
                onChange={(e) => setVerifyReason(e.target.value)}
                helperText={t('admin.case.verify_reason_help')}
              />
              <div className="flex flex-wrap gap-2">
                <Button
                  variant="secondary"
                  loading={verify.isPending}
                  onClick={() => verify.mutate({ caseId: c.id, decision: 'rejected', reason: verifyReason || undefined })}
                >
                  {t('admin.case.verify_reject')}
                </Button>
                <Button
                  loading={verify.isPending}
                  onClick={() => verify.mutate({ caseId: c.id, decision: 'verified', reason: verifyReason || undefined })}
                >
                  {t('admin.case.verify_approve')}
                </Button>
              </div>
            </div>
          )}
          {c.status === 'approved' && (
            <Button loading={publish.isPending} onClick={() => publish.mutate(c.id)}>
              {t('admin.case.publish')}
            </Button>
          )}
          {c.status === 'published' && (
            <Button variant="secondary" loading={unpublish.isPending} onClick={() => unpublish.mutate(c.id)}>
              {t('admin.case.unpublish')}
            </Button>
          )}
          <div className="border-t border-ink-200 pt-3">
            <Button
              variant="danger"
              loading={del.isPending}
              onClick={() => {
                if (confirm(t('admin.case.confirm_delete'))) {
                  del.mutate(c.id, { onSuccess: () => nav('/admin/cases') })
                }
              }}
            >
              {t('admin.case.delete_case')}
            </Button>
          </div>
        </CardBody>
      </Card>
    </div>
  )
}

function formatSize(n: number) {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}
