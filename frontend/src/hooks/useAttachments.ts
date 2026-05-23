// Attachment hooks: presign, upload (browser → R2), finalize, list,
// delete. Public listing returns only kind=public; admin listing returns
// every kind. Hidden/internal downloads use a per-request short-lived
// presigned URL.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiDelete, apiGet, apiPatch, apiPost } from '../lib/api'

export type AttachmentKind = 'public' | 'hidden' | 'internal'

export interface Attachment {
  id: string
  case_id: string
  kind: AttachmentKind
  sequence_no: number
  original_filename: string
  stored_filename: string
  storage_key: string
  public_url?: string | null
  mime_type: string
  size_bytes: number
  caption_bn?: string | null
  caption_en?: string | null
  uploaded_by?: string | null
  created_at: string
}

interface PresignResponse {
  upload_url: string
  storage_key: string
  stored_filename: string
  sequence_no: number
  presign_token: string
  expires_at: string
}

interface ListEnvelope<T> {
  data: T[]
}

/**
 * uploadFileToCase performs the full presign → PUT → finalize cycle.
 * It is exposed as a plain function (not a hook) because consumers want
 * to drive multiple uploads from a single user gesture.
 */
export async function uploadFileToCase(
  caseId: string,
  file: File,
  kind: AttachmentKind,
  onProgress?: (pct: number) => void,
): Promise<Attachment> {
  const presign = await apiPost<PresignResponse>(`/api/v1/cases/${caseId}/attachments/presign`, {
    kind,
    original_filename: file.name,
    mime_type: file.type || 'application/octet-stream',
    size_bytes: file.size,
  })

  await putWithProgress(presign.upload_url, file, onProgress)

  return apiPost<Attachment>(`/api/v1/cases/${caseId}/attachments/finalize`, {
    presign_token: presign.presign_token,
    size_bytes: file.size,
  })
}

function putWithProgress(
  url: string,
  file: File,
  onProgress?: (pct: number) => void,
): Promise<void> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('PUT', url, true)
    if (file.type) xhr.setRequestHeader('Content-Type', file.type)
    xhr.upload.onprogress = (e) => {
      if (onProgress && e.lengthComputable) onProgress((e.loaded / e.total) * 100)
    }
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) resolve()
      else reject(new Error(`Upload failed (${xhr.status}): ${xhr.responseText}`))
    }
    xhr.onerror = () => reject(new Error('Upload network error'))
    xhr.send(file)
  })
}

export function useCaseAttachments(caseId: string | undefined) {
  return useQuery<Attachment[]>({
    queryKey: ['attachments', caseId],
    queryFn: async () =>
      (await apiGet<ListEnvelope<Attachment>>(`/api/v1/cases/${caseId}/attachments`)).data,
    enabled: !!caseId,
  })
}

export function useDeleteAttachment() {
  const qc = useQueryClient()
  return useMutation<void, Error, { caseId: string; attachmentId: string }>({
    mutationFn: ({ attachmentId }) =>
      apiDelete<void>(`/api/v1/admin/attachments/${attachmentId}`),
    onSuccess: (_d, vars) => qc.invalidateQueries({ queryKey: ['attachments', vars.caseId] }),
  })
}

export function useRequestAttachmentDownload() {
  return useMutation<{ url: string; expires_at: string }, Error, string>({
    mutationFn: (id) =>
      apiPost<{ url: string; expires_at: string }>(`/api/v1/admin/attachments/${id}/download-url`),
  })
}

export function useUpdateAttachment() {
  const qc = useQueryClient()
  return useMutation<Attachment, Error, { caseId: string; attachmentId: string; payload: Partial<Pick<Attachment, 'kind' | 'caption_bn' | 'caption_en'>> }>({
    mutationFn: ({ attachmentId, payload }) =>
      apiPatch<Attachment>(`/api/v1/admin/attachments/${attachmentId}`, payload),
    onSuccess: (_d, vars) => qc.invalidateQueries({ queryKey: ['attachments', vars.caseId] }),
  })
}
