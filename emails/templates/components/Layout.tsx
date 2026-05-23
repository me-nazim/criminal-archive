import * as React from 'react'
import {
  Body,
  Container,
  Head,
  Hr,
  Html,
  Img,
  Preview,
  Section,
  Text,
} from '@react-email/components'

/**
 * Shared chrome for every Tansiq transactional email. Authored as a
 * React Email component; `react-email export` flattens this into static
 * inline-styled HTML that the Go runtime can fill with template tokens.
 */
export function Layout({
  preview,
  primaryColor = '{{ .PrimaryColor }}',
  siteName = '{{ .SiteName }}',
  logoUrl = '{{ .LogoURL }}',
  year = '{{ .Year }}',
  children,
}: {
  preview: string
  primaryColor?: string
  siteName?: string
  logoUrl?: string
  year?: string
  children: React.ReactNode
}) {
  return (
    <Html lang="en">
      <Head />
      <Preview>{preview}</Preview>
      <Body
        style={{
          margin: 0,
          padding: 0,
          background: '#f6f7f9',
          fontFamily:
            "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif",
          color: '#0f1320',
        }}
      >
        <Container
          style={{
            maxWidth: 560,
            background: '#fff',
            borderRadius: 14,
            overflow: 'hidden',
            margin: '32px auto',
            boxShadow: '0 1px 3px rgba(15,19,32,0.06)',
          }}
        >
          <Section
            style={{
              padding: '24px 32px',
              borderBottom: '1px solid #eceef2',
              fontWeight: 700,
              fontSize: 16,
            }}
          >
            {logoUrl ? (
              <Img src={logoUrl} alt={siteName} height={28} />
            ) : (
              <span style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}>
                <span
                  style={{
                    display: 'inline-block',
                    width: 24,
                    height: 24,
                    borderRadius: 6,
                    background: primaryColor,
                  }}
                />
                {siteName}
              </span>
            )}
          </Section>
          <Section style={{ padding: 32 }}>{children}</Section>
          <Hr style={{ margin: 0, borderColor: '#eceef2' }} />
          <Section
            style={{
              padding: '20px 32px',
              background: '#fafbfc',
              fontSize: 12,
              color: '#637087',
            }}
          >
            <Text style={{ margin: 0 }}>
              © {year} {siteName}.
            </Text>
          </Section>
        </Container>
      </Body>
    </Html>
  )
}
