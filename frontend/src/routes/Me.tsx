import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Container } from '../components/ui/Container'
import { Card, CardBody, CardHeader } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { Badge } from '../components/ui/Badge'
import { useAuthStore } from '../lib/auth-store'
import { useLogoutMutation } from '../hooks/useAuth'

export default function Me() {
  const { t } = useTranslation()
  const user = useAuthStore((s) => s.user)
  const logout = useLogoutMutation()
  const nav = useNavigate()

  if (!user) return null

  return (
    <Container width="reading" className="py-12">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between gap-3">
            <div>
              <h1 className="font-display text-2xl font-semibold text-ink-900">
                {user.full_name}
              </h1>
              <p className="text-sm text-ink-600">{user.email}</p>
            </div>
            <Badge tone={user.status === 'approved' ? 'success' : 'warning'}>
              {t(`role.${user.role}`)} · {t(`user_status.${user.status}`)}
            </Badge>
          </div>
        </CardHeader>
        <CardBody className="grid gap-4 text-sm sm:grid-cols-2">
          <Field label={t('me.id')} value={user.id} mono />
          <Field label={t('me.created_at')} value={new Date(user.created_at).toLocaleString()} />
          <Field
            label={t('me.last_login')}
            value={
              user.last_login_at ? new Date(user.last_login_at).toLocaleString() : t('common.never')
            }
          />
          <Field label={t('common.phone')} value={user.phone ?? '—'} />
        </CardBody>
        <CardBody className="flex flex-wrap items-center gap-2 border-t border-ink-200">
          <Button
            variant="secondary"
            onClick={() => {
              logout.mutate(undefined, {
                onSettled: () => nav('/', { replace: true }),
              })
            }}
            loading={logout.isPending}
          >
            {t('me.logout')}
          </Button>
        </CardBody>
      </Card>
    </Container>
  )
}

function Field({
  label,
  value,
  mono,
}: {
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div>
      <p className="text-xs uppercase tracking-wide text-ink-500">{label}</p>
      <p className={mono ? 'font-mono text-xs' : 'text-sm text-ink-800'}>{value}</p>
    </div>
  )
}
