// Frontend mirror of the backend `branding` settings row. The hook
// applies CSS custom properties globally so any component using
// `var(--brand-primary)` reflects admin overrides without needing the
// hook itself.

import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'

import { apiGet } from './api'

export interface Branding {
  site_name_bn: string
  site_name_en: string
  short_name: string
  tagline_bn: string
  tagline_en: string
  primary_color: string
  accent_color: string
  logo_url?: string
  favicon_url?: string
  support_email?: string
  social?: {
    twitter?: string
    facebook?: string
    youtube?: string
    github?: string
  }
}

const FALLBACK: Branding = {
  site_name_bn: 'তানসিক ইনফরমেশন পোর্টাল',
  site_name_en: 'Tansiq Information Portal',
  short_name: 'Tansiq',
  tagline_bn: 'অপরাধের সঠিক ও যাচাইকৃত পাবলিক ডকুমেন্টেশন।',
  tagline_en: 'Verified, public documentation of crimes.',
  primary_color: '#e8501f',
  accent_color: '#0f1320',
  logo_url: '',
  favicon_url: '',
  support_email: '',
  social: {},
}

/**
 * Fetch + cache the branding row. Falls back to compile-time defaults
 * when the API is unreachable so the marketing surfaces never go blank.
 */
export function useBranding(): { branding: Branding; isLoading: boolean } {
  const q = useQuery<Branding>({
    queryKey: ['branding'],
    queryFn: () => apiGet<Branding>('/api/v1/settings/branding'),
    staleTime: 5 * 60 * 1000,
    retry: 1,
  })
  const branding = q.data ?? FALLBACK

  useEffect(() => {
    const root = document.documentElement
    if (branding.primary_color) {
      root.style.setProperty('--brand-primary', branding.primary_color)
    }
    if (branding.accent_color) {
      root.style.setProperty('--brand-accent', branding.accent_color)
    }
  }, [branding.primary_color, branding.accent_color])

  return { branding, isLoading: q.isLoading }
}
