import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Bell, CheckCheck } from 'lucide-react'

import {
  useNotifications,
  useMarkNotificationRead,
  useMarkAllNotificationsRead,
  type Notification,
} from '../../hooks/useNotifications'
import { cn } from '../../lib/cn'

/**
 * The header bell. Polls /notifications every 60s and renders a small
 * popover with the most recent items + an "Mark all read" action.
 *
 * Closes on outside click and on `Escape` for keyboard users.
 */
export default function NotificationsBell() {
  const { t } = useTranslation()
  const { data } = useNotifications()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const mark = useMarkNotificationRead()
  const markAll = useMarkAllNotificationsRead()

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (!ref.current?.contains(e.target as Node)) setOpen(false)
    }
    const esc = (e: KeyboardEvent) => e.key === 'Escape' && setOpen(false)
    document.addEventListener('mousedown', handler)
    document.addEventListener('keydown', esc)
    return () => {
      document.removeEventListener('mousedown', handler)
      document.removeEventListener('keydown', esc)
    }
  }, [])

  const unread = data?.unread ?? 0
  const items = data?.data ?? []

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="relative inline-flex h-9 w-9 items-center justify-center rounded-md border border-ink-300 text-ink-800 hover:bg-ink-100"
        aria-label={t('notifications.aria_open') ?? 'Notifications'}
        aria-haspopup="menu"
        aria-expanded={open}
      >
        <Bell className="h-4 w-4" aria-hidden />
        {unread > 0 && (
          <span
            className="absolute -top-1 -right-1 inline-flex min-w-[18px] items-center justify-center rounded-full px-1 text-[10px] font-semibold text-white"
            style={{ background: 'var(--brand-primary)' }}
          >
            {unread > 99 ? '99+' : unread}
          </span>
        )}
      </button>
      {open && (
        <div className="absolute right-0 top-11 z-40 w-[22rem] max-w-[90vw] overflow-hidden rounded-xl border border-ink-200 bg-white shadow-elevated">
          <div className="flex items-center justify-between border-b border-ink-200 px-4 py-3">
            <p className="text-sm font-semibold text-ink-900">{t('notifications.title')}</p>
            <button
              type="button"
              disabled={unread === 0}
              onClick={() => markAll.mutate()}
              className={cn(
                'inline-flex items-center gap-1 text-xs font-medium',
                unread === 0 ? 'text-ink-400' : 'text-ink-700 hover:text-ink-900',
              )}
            >
              <CheckCheck className="h-3.5 w-3.5" aria-hidden />
              {t('notifications.mark_all')}
            </button>
          </div>
          <div className="max-h-[60vh] overflow-y-auto">
            {items.length === 0 && (
              <div className="px-4 py-10 text-center text-sm text-ink-500">
                {t('notifications.empty')}
              </div>
            )}
            {items.map((n) => (
              <NotificationRow
                key={n.id}
                n={n}
                onClick={() => {
                  if (!n.read_at) mark.mutate(n.id)
                  setOpen(false)
                }}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function NotificationRow({ n, onClick }: { n: Notification; onClick: () => void }) {
  const inner = (
    <div
      className={cn(
        'flex items-start gap-3 border-b border-ink-100 px-4 py-3 last:border-b-0 hover:bg-ink-50',
        !n.read_at && 'bg-brand-50/40',
      )}
    >
      <span
        className={cn(
          'mt-1 inline-block h-2 w-2 shrink-0 rounded-full',
          n.read_at ? 'bg-ink-300' : 'bg-brand-500',
        )}
        aria-hidden
      />
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium text-ink-900">{n.title}</p>
        {n.body && <p className="mt-0.5 line-clamp-2 text-xs text-ink-600">{n.body}</p>}
        <p className="mt-1 text-[11px] text-ink-500">
          {new Date(n.created_at).toLocaleString()}
        </p>
      </div>
    </div>
  )
  if (n.link) {
    return (
      <Link to={n.link} onClick={onClick} className="block">
        {inner}
      </Link>
    )
  }
  return (
    <button type="button" onClick={onClick} className="block w-full text-left">
      {inner}
    </button>
  )
}
