import { forwardRef, useId, type InputHTMLAttributes, type ReactNode } from 'react'
import { cn } from '../../lib/cn'

export interface TextFieldProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string
  helperText?: string
  errorText?: string
  leftAddon?: ReactNode
  rightAddon?: ReactNode
  /** Marks the field visually with a red asterisk. Does not affect HTML required attribute. */
  showRequired?: boolean
}

export const TextField = forwardRef<HTMLInputElement, TextFieldProps>(function TextField(
  {
    id,
    label,
    helperText,
    errorText,
    leftAddon,
    rightAddon,
    showRequired,
    className,
    disabled,
    ...rest
  },
  ref,
) {
  const auto = useId()
  const inputId = id ?? `f-${auto}`
  const helpId = helperText ? `${inputId}-help` : undefined
  const errId = errorText ? `${inputId}-err` : undefined

  return (
    <div className={cn('flex flex-col gap-1.5', disabled && 'opacity-60')}>
      {label && (
        <label htmlFor={inputId} className="text-sm font-medium text-ink-800">
          {label}
          {showRequired && <span className="ml-0.5 text-red-600" aria-hidden>*</span>}
        </label>
      )}
      <div
        className={cn(
          'flex items-center rounded-md border bg-white shadow-sm',
          errorText
            ? 'border-red-400 focus-within:border-red-500 focus-within:ring-red-500'
            : 'border-ink-300 focus-within:border-brand-500 focus-within:ring-brand-500',
          'focus-within:ring-2 focus-within:ring-offset-0',
        )}
      >
        {leftAddon && <span className="pl-3 text-ink-500">{leftAddon}</span>}
        <input
          id={inputId}
          ref={ref}
          aria-invalid={errorText ? 'true' : undefined}
          aria-describedby={[helpId, errId].filter(Boolean).join(' ') || undefined}
          disabled={disabled}
          className={cn(
            'h-10 w-full bg-transparent px-3 text-sm text-ink-900 outline-none placeholder:text-ink-400',
            'disabled:cursor-not-allowed',
            className,
          )}
          {...rest}
        />
        {rightAddon && <span className="pr-3 text-ink-500">{rightAddon}</span>}
      </div>
      {errorText ? (
        <p id={errId} className="text-xs text-red-600">
          {errorText}
        </p>
      ) : helperText ? (
        <p id={helpId} className="text-xs text-ink-500">
          {helperText}
        </p>
      ) : null}
    </div>
  )
})
