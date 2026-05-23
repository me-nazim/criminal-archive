// Public + authenticated hooks for the persons resource. Mutation hooks
// invalidate the relevant query keys so list views stay fresh.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiGet, apiPatch, apiPost } from '../lib/api'

export interface Person {
  id: string
  slug: string
  full_name_bn?: string | null
  full_name_en?: string | null
  aliases: string[]
  primary_type: 'victim' | 'accused' | 'witness' | 'other'
  gender?: string | null
  date_of_birth?: string | null
  photo_url?: string | null
  occupation?: string | null
  organization?: string | null
  designation?: string | null
  country_id?: number | null
  division_id?: number | null
  district_id?: number | null
  upazila_id?: number | null
  address_line?: string | null
  public_bio_bn?: string | null
  public_bio_en?: string | null
  internal_notes?: string | null
  is_anonymous: boolean
  status: string
  case_count?: number
  created_at: string
  updated_at: string
}

interface ListEnvelope<T> {
  data: T[]
}

export function usePublicPersons(params: { primary_type?: string; q?: string } = {}) {
  const qs = new URLSearchParams()
  if (params.primary_type) qs.set('primary_type', params.primary_type)
  if (params.q) qs.set('q', params.q)
  const suffix = qs.toString() ? `?${qs.toString()}` : ''
  return useQuery<Person[]>({
    queryKey: ['persons', 'public', params],
    queryFn: async () => (await apiGet<ListEnvelope<Person>>(`/api/v1/persons${suffix}`)).data,
    staleTime: 60_000,
  })
}

export function usePerson(slugOrID: string | undefined) {
  return useQuery<Person>({
    queryKey: ['person', slugOrID],
    queryFn: () => apiGet<Person>(`/api/v1/persons/${slugOrID}`),
    enabled: !!slugOrID,
  })
}

export function useMyPersons() {
  return useQuery<Person[]>({
    queryKey: ['persons', 'mine'],
    queryFn: async () => (await apiGet<ListEnvelope<Person>>(`/api/v1/me/persons`)).data,
  })
}

export function useAdminPersons(params: { status?: string; q?: string } = {}) {
  const qs = new URLSearchParams()
  if (params.status) qs.set('status', params.status)
  if (params.q) qs.set('q', params.q)
  const suffix = qs.toString() ? `?${qs.toString()}` : ''
  return useQuery<Person[]>({
    queryKey: ['persons', 'admin', params],
    queryFn: async () =>
      (await apiGet<ListEnvelope<Person>>(`/api/v1/admin/persons${suffix}`)).data,
  })
}

export function useCreatePerson() {
  const qc = useQueryClient()
  return useMutation<Person, Error, Partial<Person>>({
    mutationFn: (payload) => apiPost<Person>('/api/v1/persons', payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['persons'] })
    },
  })
}

export function useUpdatePerson(id: string) {
  const qc = useQueryClient()
  return useMutation<Person, Error, Partial<Person>>({
    mutationFn: (payload) => apiPatch<Person>(`/api/v1/persons/${id}`, payload),
    onSuccess: (data) => {
      qc.invalidateQueries({ queryKey: ['persons'] })
      qc.setQueryData(['person', id], data)
      qc.setQueryData(['person', data.slug], data)
    },
  })
}

export function useApprovePerson() {
  const qc = useQueryClient()
  return useMutation<Person, Error, string>({
    mutationFn: (id) => apiPost<Person>(`/api/v1/admin/persons/${id}/approve`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['persons'] }),
  })
}
