// Data hooks for the public reference endpoints. These are heavily
// cached; staleTime/gcTime are set on the global QueryClient so we
// keep the hooks themselves trivial.

import { useQuery } from '@tanstack/react-query'
import { apiGet } from '../lib/api'

interface BaseLocaleName {
  id: number
  name_en: string
  name_bn: string | null
}

export interface Country extends BaseLocaleName {
  iso2: string
  iso3: string
  phone_code: string | null
}

export interface Division extends BaseLocaleName {
  country_id: number
}

export interface District extends BaseLocaleName {
  division_id: number
}

export interface Upazila extends BaseLocaleName {
  district_id: number
}

export interface CrimeType {
  id: number
  slug: string
  name_en: string
  name_bn: string
  description?: string | null
  severity: number
}

interface Envelope<T> {
  data: T
}

export function useCountries() {
  return useQuery<Country[]>({
    queryKey: ['countries'],
    queryFn: async () => (await apiGet<Envelope<Country[]>>('/api/v1/locations/countries')).data,
    staleTime: 24 * 60 * 60_000,
  })
}

export function useDivisions(countryId: number | null) {
  return useQuery<Division[]>({
    queryKey: ['divisions', countryId],
    queryFn: async () => {
      if (!countryId) return []
      return (
        await apiGet<Envelope<Division[]>>(`/api/v1/locations/divisions?country_id=${countryId}`)
      ).data
    },
    enabled: !!countryId,
    staleTime: 24 * 60 * 60_000,
  })
}

export function useDistricts(divisionId: number | null) {
  return useQuery<District[]>({
    queryKey: ['districts', divisionId],
    queryFn: async () => {
      if (!divisionId) return []
      return (
        await apiGet<Envelope<District[]>>(`/api/v1/locations/districts?division_id=${divisionId}`)
      ).data
    },
    enabled: !!divisionId,
    staleTime: 24 * 60 * 60_000,
  })
}

export function useUpazilas(districtId: number | null) {
  return useQuery<Upazila[]>({
    queryKey: ['upazilas', districtId],
    queryFn: async () => {
      if (!districtId) return []
      return (
        await apiGet<Envelope<Upazila[]>>(`/api/v1/locations/upazilas?district_id=${districtId}`)
      ).data
    },
    enabled: !!districtId,
    staleTime: 24 * 60 * 60_000,
  })
}

export function useCrimeTypes() {
  return useQuery<CrimeType[]>({
    queryKey: ['crime-types'],
    queryFn: async () => (await apiGet<Envelope<CrimeType[]>>('/api/v1/crime-types')).data,
    staleTime: 60 * 60_000,
  })
}

/**
 * pickName returns the locale-preferred name with a graceful fallback:
 *  - if locale starts with "bn" and a Bangla name exists, use it;
 *  - otherwise prefer English;
 *  - finally fall back to whatever non-empty value is available.
 */
export function pickName(
  row: { name_en: string; name_bn: string | null },
  locale: string,
): string {
  if (locale.startsWith('bn') && row.name_bn) return row.name_bn
  return row.name_en || row.name_bn || ''
}
