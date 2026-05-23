import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from 'react'
import { Loader2 } from 'lucide-react'
import { cn } from '../../lib/cn'

type Variant = 'primary' | 'secondary' | 'ghost' | 'danger'
type Size = 'sm' | 'md' | 'lg'

const VARIANT: Record<Variant, string> = {
  primary:
    'bg-ink-900 text-white hover:bg-ink-800 active:bg-ink-700 disabled:bg-ink-300 disabled:text-ink-500',
  secondary:
    'border border-ink-300 bg-white text-ink-800 hover:bg-ink-100 active:bg-ink-200 disabled:bg-ink-50 disabled:text-ink-400 disabled:border-ink-200',
  ghost:
    'bg-transparent text-ink-700 hover:bg-ink-100 disabled:text-ink-400',
  danger:
    'bg-red-600 text-white hover:bg-red-700 active:bg-red-800 disabled:bg-red-300',
}

const SIZE: Record<Size, string> = {
  sm: 'h-8 px-3 text-xs',
  md: 'h-10 px-4 text-sm',
  lg: 'h-12 px-5 text-base',
}

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  size?: Size
  loading?: boolean
  leftIcon?: ReactNode
  rightIcon?: ReactNode
  fullWidth?: boolean
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  {
    className,
    variant = 'primary',
    size = 'md',
    loading = false,
    disabled,
    leftIcon,
    rightIcon,
    fullWidth,
    children,
    ...rest
  },
  ref,
) {
  return (
    <button
      ref={ref}
      type={rest.type ?? 'button'}
      disabled={disabled || loading}
      className={cn(
        'inline-flex items-center justify-center gap-2 rounded-md font-medium',
        'transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 focus-visible:ring-offset-2',
        'disabled:cursor-not-allowed',
        VARIANT[variant],
        SIZE[size],
        fullWidth && 'w-full',
        className,
      )}
      {...rest}
    >
      {loading ? (
        <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
      ) : (
        leftIcon && <span aria-hidden>{leftIcon}</span>
      )}
      {children}
      {!loading && rightIcon && <span aria-hidden>{rightIcon}</span>}
    </button>
  )
})
