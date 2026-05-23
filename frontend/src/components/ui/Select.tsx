import { forwardRef, useId, type SelectHTMLAttributes, type ReactNode } from 'react'
import { cn } from '../../lib/cn'

export interface SelectOption {
  value: string | number
  label: string
  disabled?: boolean
}

export interface SelectProps
  extends Omit<SelectHTMLAttributes<HTMLSelectElement>, 'children'> {
  label?: string
  helperText?: string
  errorText?: string
  options: SelectOption[]
  placeholder?: string
  showRequired?: boolean
  /** Optional content rendered above the placeholder (e.g. clear option). */
  prepend?: ReactNode
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(function Select(
  {
    id,
    label,
    helperText,
    errorText,
    options,
    placeholder,
    showRequired,
    prepend,
    className,
    disabled,
    ...rest
  },
  ref,
) {
  const auto = useId()
  const inputId = id ?? `s-${auto}`
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
      <select
        id={inputId}
        ref={ref}
        aria-invalid={errorText ? 'true' : undefined}
        aria-describedby={[helpId, errId].filter(Boolean).join(' ') || undefined}
        disabled={disabled}
        className={cn(
          'h-10 rounded-md border bg-white px-3 text-sm text-ink-900 shadow-sm focus:outline-none focus:ring-2 focus:ring-offset-0',
          errorText
            ? 'border-red-400 focus:border-red-500 focus:ring-red-500'
            : 'border-ink-300 focus:border-brand-500 focus:ring-brand-500',
          'disabled:cursor-not-allowed',
          className,
        )}
        {...rest}
      >
        {placeholder && (
          <option value="" disabled hidden={!rest.value && rest.value !== 0}>
            {placeholder}
          </option>
        )}
        {prepend}
        {options.map((o) => (
          <option key={String(o.value)} value={o.value} disabled={o.disabled}>
            {o.label}
          </option>
        ))}
      </select>
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
