import * as React from 'react'
import { Button, Heading, Text } from '@react-email/components'
import { Layout } from './components/Layout'

/**
 * `user.welcome` — sent right after a self-registration succeeds.
 *
 * Variables:
 *   FullName - recipient's full name
 *   SiteName / SiteURL / PrimaryColor - branding (provided by Go runtime)
 */
export default function UserWelcome() {
  return (
    <Layout preview="Your Tansiq account is awaiting admin review">
      <Heading style={{ fontSize: 22, margin: '0 0 16px' }}>
        Welcome, {'{{ .FullName }}'}.
      </Heading>
      <Text style={{ fontSize: 15, lineHeight: '1.6', margin: '0 0 14px' }}>
        Thank you for registering with <strong>{'{{ .SiteName }}'}</strong>. Your
        account is now in our review queue. An admin will verify your details
        before activating your access.
      </Text>
      <Text style={{ fontSize: 15, lineHeight: '1.6', margin: '0 0 14px' }}>
        You will receive a follow-up email once your account is approved. In the
        meantime, you can browse public, verified cases on the portal.
      </Text>
      <Button
        href={'{{ .SiteURL }}'}
        style={{
          background: '{{ .PrimaryColor }}',
          color: '#fff',
          padding: '12px 22px',
          borderRadius: 8,
          fontWeight: 600,
        }}
      >
        Open the portal
      </Button>
      <Text style={{ fontSize: 13, color: '#637087', margin: '24px 0 0' }}>
        If you did not register with {'{{ .SiteName }}'}, please ignore this
        email — no account will be activated without admin approval.
      </Text>
    </Layout>
  )
}
