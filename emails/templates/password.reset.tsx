import * as React from 'react'
import { Button, Heading, Section, Text } from '@react-email/components'
import { Layout } from './components/Layout'

/**
 * `password.reset` — sent when the user requests a password reset.
 *
 * Variables:
 *   FullName, ResetURL, ExpiresInMinutes
 */
export default function PasswordReset() {
  return (
    <Layout preview="Reset your Tansiq password">
      <Heading style={{ fontSize: 22, margin: '0 0 16px' }}>
        Reset your password
      </Heading>
      <Text style={{ fontSize: 15, lineHeight: '1.6', margin: '0 0 14px' }}>
        We received a request to reset the password for your account on{' '}
        <strong>{'{{ .SiteName }}'}</strong>. The link below will expire in{' '}
        {'{{ .ExpiresInMinutes }}'} minutes.
      </Text>
      <Button
        href={'{{ .ResetURL }}'}
        style={{
          background: '{{ .PrimaryColor }}',
          color: '#fff',
          padding: '12px 22px',
          borderRadius: 8,
          fontWeight: 600,
        }}
      >
        Reset my password
      </Button>
      <Section style={{ marginTop: 24 }}>
        <Text style={{ fontSize: 13, color: '#637087', margin: '0 0 6px' }}>
          If the button doesn't work, paste this URL into your browser:
        </Text>
        <Text
          style={{
            fontSize: 12,
            color: '#4d586d',
            wordBreak: 'break-all',
            background: '#f6f7f9',
            padding: '10px 12px',
            borderRadius: 6,
            margin: 0,
          }}
        >
          {'{{ .ResetURL }}'}
        </Text>
      </Section>
      <Text style={{ fontSize: 13, color: '#637087', margin: '24px 0 0' }}>
        If you did not request a password reset, you can safely ignore this
        email — your password will remain unchanged.
      </Text>
    </Layout>
  )
}
