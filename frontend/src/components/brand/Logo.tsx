import { cn } from '../../lib/cn'

interface LogoProps {
  /** When true, only render the bookmark mark (no wordmark text). */
  markOnly?: boolean
  /** Override CSS class on the wrapper (sizing, layout). */
  className?: string
  /** Optional URL to a custom logo configured by the admin. */
  customSrc?: string
  /** Render the wordmark in Bengali instead of Latin. */
  bengali?: boolean
  /** Override label for screen readers. */
  label?: string
}

/**
 * The Tansiq mark. A bookmark / record icon — a vertical pin grounded
 * on a horizontal line of evidence — coloured in the portal's primary
 * accent. Composes with the wordmark unless `markOnly` is true.
 *
 * The component is fully accessible: when no custom logo is set we
 * render an inline SVG so it inherits `currentColor` and adapts to dark
 * surfaces; when an admin uploads a logo URL we use an <img>.
 */
export function Logo({ markOnly, className, customSrc, bengali, label }: LogoProps) {
  const a11yLabel = label ?? 'Tansiq Information Portal'
  if (customSrc) {
    return (
      <img
        src={customSrc}
        alt={a11yLabel}
        className={cn('h-8 w-auto', className)}
      />
    )
  }
  return (
    <span
      className={cn('inline-flex items-center gap-2.5 leading-none', className)}
      aria-label={a11yLabel}
    >
      <Mark />
      {!markOnly && (
        <span className="flex flex-col">
          <span className="font-display text-[1.05rem] font-semibold tracking-tight text-ink-900">
            {bengali ? 'তানসিক' : 'Tansiq'}
          </span>
          <span className="text-[0.625rem] font-semibold uppercase tracking-[0.18em] text-ink-500">
            {bengali ? 'ইনফরমেশন পোর্টাল' : 'Information Portal'}
          </span>
        </span>
      )}
    </span>
  )
}

function Mark() {
  // The mark sits in a 32×32 box. The pin colour is driven by the
  // CSS variable so admin theming of `--brand-primary` flows through.
  return (
    <svg
      width="32"
      height="32"
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden
      className="shrink-0"
    >
      <rect x="0" y="0" width="32" height="32" rx="8" className="fill-ink-900" />
      <path
        d="M8 11h16M16 11v13"
        stroke="white"
        strokeWidth="2.4"
        strokeLinecap="round"
      />
      <circle cx="16" cy="24" r="2.2" style={{ fill: 'var(--brand-primary)' }} />
    </svg>
  )
}
