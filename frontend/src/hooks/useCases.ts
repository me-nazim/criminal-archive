// Hooks around the cases resource. The shape mirrors the backend
// response: detail returns { case, persons, timeline, news_sources }.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiDelete, apiGet, apiPatch, apiPost } from '../lib/api'

export interface CaseRow {
  id: string
  case_number: string
  slug: string
  title_bn: string
  title_en?: string | null
  summary_bn?: string | null
  summary_en?: string | null
  description_bn?: string | null
  description_en?: string | null
  internal_notes?: string | null
  incident_date?: string | null
  incident_time?: string | null
  country_id?: number | null
  division_id?: number | null
  district_id?: number | null
  upazila_id?: number | null
  location_text?: string | null
  crime_type_id?: number | null
  case_status?: string | null
  severity?: number | null
  cover_image_url?: string | null
  tags: string[]
  status: string
  view_count: number
  download_count: number
  published_at?: string | null
  created_at: string
  updated_at: string
}

export interface CasePersonLink {
  person_id: string
  person_slug: string
  role: string
  is_anonymous: boolean
  name_bn?: string | null
  name_en?: string | null
  photo_url?: string | null
  notes?: string | null
}

export interface TimelineEvent {
  id: string
  case_id: string
  event_date: string
  event_time?: string | null
  title_bn: string
  title_en?: string | null
  description_bn?: string | null
  description_en?: string | null
  source_url?: string | null
  is_internal: boolean
  created_at: string
}

export interface NewsSource {
  id: string
  case_id: string
  url: string
  title?: string | null
  source_name?: string | null
  published_at?: string | null
  archived_url?: string | null
  created_at: string
}

export interface CaseDetail {
  case: CaseRow
  persons: CasePersonLink[]
  timeline: TimelineEvent[]
  news_sources: NewsSource[]
}

interface ListEnvelope<T> {
  data: T[]
}

export interface CaseFilters {
  q?: string
  country_id?: number
  division_id?: number
  district_id?: number
  upazila_id?: number
  crime_type_id?: number
  year?: number
  tag?: string
  sort?: 'incident_desc' | 'published_desc'
  limit?: number
}

function toQuery(p: CaseFilters): string {
  const q = new URLSearchParams()
  Object.entries(p).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== '') q.set(k, String(v))
  })
  const s = q.toString()
  return s ? `?${s}` : ''
}

export function usePublicCases(filters: CaseFilters = {}) {
  return useQuery<CaseRow[]>({
    queryKey: ['cases', 'public', filters],
    queryFn: async () =>
      (await apiGet<ListEnvelope<CaseRow>>(`/api/v1/cases${toQuery(filters)}`)).data,
    staleTime: 30_000,
  })
}

export function usePublicCase(key: string | undefined) {
  return useQuery<CaseDetail>({
    queryKey: ['case', key],
    queryFn: () => apiGet<CaseDetail>(`/api/v1/cases/${key}`),
    enabled: !!key,
  })
}

export function useMyCases() {
  return useQuery<CaseRow[]>({
    queryKey: ['cases', 'mine'],
    queryFn: async () => (await apiGet<ListEnvelope<CaseRow>>('/api/v1/me/cases')).data,
  })
}

export function useMyCase(id: string | undefined) {
  return useQuery<CaseDetail>({
    queryKey: ['case', 'mine', id],
    queryFn: () => apiGet<CaseDetail>(`/api/v1/me/cases/${id}`),
    enabled: !!id,
  })
}

export function useAdminCases(filters: CaseFilters & { status?: string } = {}) {
  const q = new URLSearchParams()
  Object.entries(filters).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== '') q.set(k, String(v))
  })
  const suffix = q.toString() ? `?${q.toString()}` : ''
  return useQuery<CaseRow[]>({
    queryKey: ['cases', 'admin', filters],
    queryFn: async () => (await apiGet<ListEnvelope<CaseRow>>(`/api/v1/admin/cases${suffix}`)).data,
  })
}

export function useAdminCase(id: string | undefined) {
  return useQuery<CaseDetail>({
    queryKey: ['case', 'admin', id],
    queryFn: () => apiGet<CaseDetail>(`/api/v1/admin/cases/${id}`),
    enabled: !!id,
  })
}

export function useCasesForPerson(slugOrID: string | undefined) {
  return useQuery<CaseRow[]>({
    queryKey: ['cases', 'person', slugOrID],
    queryFn: async () =>
      (await apiGet<ListEnvelope<CaseRow>>(`/api/v1/persons/${slugOrID}/cases`)).data,
    enabled: !!slugOrID,
  })
}

// ---------- mutations ----------

export function useCreateCase() {
  const qc = useQueryClient()
  return useMutation<CaseRow, Error, Partial<CaseRow>>({
    mutationFn: (payload) => apiPost<CaseRow>('/api/v1/cases', payload),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['cases'] }),
  })
}

export function usePatchCase(id: string) {
  const qc = useQueryClient()
  return useMutation<CaseRow, Error, Partial<CaseRow>>({
    mutationFn: (payload) => apiPatch<CaseRow>(`/api/v1/cases/${id}`, payload),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['cases'] }),
  })
}

function caseAction(verb: 'submit' | 'publish' | 'unpublish') {
  // eslint-disable-next-line react-hooks/rules-of-hooks
  return () => {
    const qc = useQueryClient()
    return useMutation<CaseRow, Error, string>({
      mutationFn: (id) => apiPost<CaseRow>(`/api/v1/cases/${id}/${verb}`),
      onSuccess: () => qc.invalidateQueries({ queryKey: ['cases'] }),
    })
  }
}

export const useSubmitCase = caseAction('submit')

export function usePublishCase() {
  const qc = useQueryClient()
  return useMutation<CaseRow, Error, string>({
    mutationFn: (id) => apiPost<CaseRow>(`/api/v1/admin/cases/${id}/publish`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['cases'] }),
  })
}

export function useUnpublishCase() {
  const qc = useQueryClient()
  return useMutation<CaseRow, Error, string>({
    mutationFn: (id) => apiPost<CaseRow>(`/api/v1/admin/cases/${id}/unpublish`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['cases'] }),
  })
}

export function useAssignVerifier() {
  const qc = useQueryClient()
  return useMutation<CaseRow, Error, { caseId: string; assigneeId: string }>({
    mutationFn: ({ caseId, assigneeId }) =>
      apiPost<CaseRow>(`/api/v1/admin/cases/${caseId}/assign`, { assignee_id: assigneeId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['cases'] }),
  })
}

export function useVerifyCase() {
  const qc = useQueryClient()
  return useMutation<CaseRow, Error, { caseId: string; decision: 'verified' | 'rejected'; reason?: string }>({
    mutationFn: ({ caseId, decision, reason }) =>
      apiPost<CaseRow>(`/api/v1/admin/cases/${caseId}/verify`, { decision, reason }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['cases'] }),
  })
}

export function useDeleteCase() {
  const qc = useQueryClient()
  return useMutation<void, Error, string>({
    mutationFn: (id) => apiDelete<void>(`/api/v1/admin/cases/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['cases'] }),
  })
}

export function useAddCasePerson() {
  const qc = useQueryClient()
  return useMutation<void, Error, { caseId: string; personId: string; role: string; notes?: string }>({
    mutationFn: ({ caseId, personId, role, notes }) =>
      apiPost<void>(`/api/v1/cases/${caseId}/persons`, { person_id: personId, role, notes }),
    onSuccess: (_d, vars) => {
      qc.invalidateQueries({ queryKey: ['case', vars.caseId] })
      qc.invalidateQueries({ queryKey: ['case', 'mine', vars.caseId] })
      qc.invalidateQueries({ queryKey: ['case', 'admin', vars.caseId] })
    },
  })
}

export function useRemoveCasePerson() {
  const qc = useQueryClient()
  return useMutation<void, Error, { caseId: string; personId: string; role: string }>({
    mutationFn: ({ caseId, personId, role }) =>
      apiDelete<void>(`/api/v1/cases/${caseId}/persons/${personId}/${role}`),
    onSuccess: (_d, vars) => {
      qc.invalidateQueries({ queryKey: ['case', vars.caseId] })
    },
  })
}

export function useAddTimelineEvent() {
  const qc = useQueryClient()
  return useMutation<TimelineEvent, Error, { caseId: string } & Omit<TimelineEvent, 'id' | 'case_id' | 'created_at'>>({
    mutationFn: ({ caseId, ...payload }) =>
      apiPost<TimelineEvent>(`/api/v1/cases/${caseId}/timeline`, payload),
    onSuccess: (_d, vars) => qc.invalidateQueries({ queryKey: ['case', vars.caseId] }),
  })
}

export function useAddNewsSource() {
  const qc = useQueryClient()
  return useMutation<NewsSource, Error, { caseId: string } & Omit<NewsSource, 'id' | 'case_id' | 'created_at'>>({
    mutationFn: ({ caseId, ...payload }) =>
      apiPost<NewsSource>(`/api/v1/cases/${caseId}/news-sources`, payload),
    onSuccess: (_d, vars) => qc.invalidateQueries({ queryKey: ['case', vars.caseId] }),
  })
}
