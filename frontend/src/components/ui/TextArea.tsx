import { forwardRef, useId, type TextareaHTMLAttributes } from 'react'
import { cn } from '../../lib/cn'

export interface TextAreaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string
  helperText?: string
  errorText?: string
  showRequired?: boolean
}

export const TextArea = forwardRef<HTMLTextAreaElement, TextAreaProps>(function TextArea(
  { id, label, helperText, errorText, showRequired, className, disabled, rows = 4, ...rest },
  ref,
) {
  const auto = useId()
  const inputId = id ?? `t-${auto}`
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
      <textarea
        id={inputId}
        ref={ref}
        rows={rows}
        aria-invalid={errorText ? 'true' : undefined}
        aria-describedby={[helpId, errId].filter(Boolean).join(' ') || undefined}
        disabled={disabled}
        className={cn(
          'rounded-md border bg-white px-3 py-2 text-sm text-ink-900 shadow-sm focus:outline-none focus:ring-2 focus:ring-offset-0',
          errorText
            ? 'border-red-400 focus:border-red-500 focus:ring-red-500'
            : 'border-ink-300 focus:border-brand-500 focus:ring-brand-500',
          'disabled:cursor-not-allowed',
          className,
        )}
        {...rest}
      />
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
