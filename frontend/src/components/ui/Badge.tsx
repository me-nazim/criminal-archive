import { type ReactNode } from 'react'
import { cn } from '../../lib/cn'

type Tone = 'neutral' | 'success' | 'warning' | 'danger' | 'info' | 'brand'

const TONE: Record<Tone, string> = {
  neutral: 'bg-ink-100 text-ink-700',
  success: 'bg-green-100 text-green-800',
  warning: 'bg-amber-100 text-amber-800',
  danger: 'bg-red-100 text-red-800',
  info: 'bg-blue-100 text-blue-800',
  brand: 'bg-brand-100 text-brand-800',
}

export interface BadgeProps {
  tone?: Tone
  children: ReactNode
  className?: string
}

export function Badge({ tone = 'neutral', children, className }: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
        TONE[tone],
        className,
      )}
    >
      {children}
    </span>
  )
}
