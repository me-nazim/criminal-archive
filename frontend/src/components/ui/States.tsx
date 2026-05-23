import { type ReactNode } from 'react'
import { Loader2, AlertTriangle, Inbox } from 'lucide-react'
import { Button } from './Button'
import { cn } from '../../lib/cn'

interface BaseProps {
  className?: string
  title?: ReactNode
  message?: ReactNode
  icon?: ReactNode
}

export function LoadingState({ className, title, message, icon }: BaseProps) {
  return (
    <div
      role="status"
      aria-live="polite"
      className={cn('flex flex-col items-center gap-3 py-12 text-center text-ink-500', className)}
    >
      {icon ?? <Loader2 className="h-6 w-6 animate-spin text-ink-400" aria-hidden />}
      <div>
        {title && <p className="text-sm font-medium text-ink-700">{title}</p>}
        {message && <p className="text-xs text-ink-500">{message}</p>}
      </div>
    </div>
  )
}

export function EmptyState({
  className,
  title,
  message,
  icon,
  action,
}: BaseProps & { action?: ReactNode }) {
  return (
    <div
      className={cn(
        'flex flex-col items-center gap-3 rounded-lg border border-dashed border-ink-200 bg-ink-50 px-6 py-12 text-center',
        className,
      )}
    >
      {icon ?? <Inbox className="h-8 w-8 text-ink-400" aria-hidden />}
      {title && <p className="text-sm font-medium text-ink-800">{title}</p>}
      {message && <p className="max-w-prose text-xs text-ink-500">{message}</p>}
      {action && <div className="mt-2">{action}</div>}
    </div>
  )
}

export function ErrorState({
  className,
  title,
  message,
  icon,
  onRetry,
  retryLabel = 'Try again',
}: BaseProps & { onRetry?: () => void; retryLabel?: string }) {
  return (
    <div
      role="alert"
      className={cn(
        'flex flex-col items-center gap-3 rounded-lg border border-red-200 bg-red-50 px-6 py-10 text-center',
        className,
      )}
    >
      {icon ?? <AlertTriangle className="h-7 w-7 text-red-500" aria-hidden />}
      <div className="space-y-1">
        <p className="text-sm font-medium text-red-800">{title ?? 'Something went wrong.'}</p>
        {message && <p className="text-xs text-red-700">{message}</p>}
      </div>
      {onRetry && (
        <Button variant="secondary" size="sm" onClick={onRetry}>
          {retryLabel}
        </Button>
      )}
    </div>
  )
}
