-- =====================================================================
-- Tansiq Information Portal — App settings, email outbox,
-- password resets, in-app notifications.
-- =====================================================================

BEGIN;

-- ---------------------------------------------------------------------
-- APP SETTINGS  (typed JSONB blobs, one row per logical key)
-- Sensitive fields inside `value` are wrapped as
--   {"__enc": true, "ciphertext": "<base64>"} and decrypted on read.
-- ---------------------------------------------------------------------
CREATE TABLE app_settings (
    key         TEXT PRIMARY KEY,
    value       JSONB NOT NULL DEFAULT '{}'::jsonb,
    description TEXT,
    updated_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Default rows so the admin UI always has a baseline to render.
INSERT INTO app_settings (key, value, description) VALUES
    ('branding', jsonb_build_object(
        'site_name_bn',  'তানসিক ইনফরমেশন পোর্টাল',
        'site_name_en',  'Tansiq Information Portal',
        'short_name',    'Tansiq',
        'tagline_bn',    'অপরাধের সঠিক ও যাচাইকৃত পাবলিক ডকুমেন্টেশন।',
        'tagline_en',    'Verified, public documentation of crimes.',
        'primary_color', '#e8501f',
        'accent_color',  '#0f1320',
        'logo_url',      '',
        'favicon_url',   '',
        'support_email', '',
        'social', jsonb_build_object(
            'twitter',  '',
            'facebook', '',
            'youtube',  '',
            'github',   ''
        )
    ), 'Public branding (site names, colours, logos, social).'),
    ('email', jsonb_build_object(
        'enabled',     false,
        'provider',    'smtp',
        'from_address','no-reply@example.com',
        'from_name',   'Tansiq Information Portal',
        'reply_to',    '',
        'smtp', jsonb_build_object(
            'host',     '',
            'port',     587,
            'username', '',
            'password', '',
            'starttls', true,
            'use_tls',  false
        ),
        'resend',       jsonb_build_object('api_key', ''),
        'elastic_mail', jsonb_build_object('api_key', '', 'base_url', 'https://api.elasticemail.com/v4')
    ), 'Email provider configuration. Secret fields are encrypted.'),
    ('storage', jsonb_build_object(
        'enabled',          true,
        'driver',           'minio',
        'bucket',           'tansiq-archive',
        'region',           'auto',
        'endpoint',         'http://minio:9000',
        'access_key',       '',
        'secret_key',       '',
        'public_base_url',  'http://localhost:9000/tansiq-archive',
        'force_path_style', true
    ), 'Object storage configuration (R2, AWS S3, MinIO, S3-compatible). Secret fields are encrypted.'),
    ('features', jsonb_build_object(
        'allow_public_registration', true,
        'require_email_verification', false,
        'maintenance_mode',          false,
        'maintenance_message_bn',    '',
        'maintenance_message_en',    '',
        'banner_enabled',            false,
        'banner_level',              'info',
        'banner_message_bn',         '',
        'banner_message_en',         ''
    ), 'Feature toggles, banner, maintenance mode.')
ON CONFLICT (key) DO NOTHING;

-- ---------------------------------------------------------------------
-- EMAIL OUTBOX  (durable queue + audit trail of every send attempt)
-- ---------------------------------------------------------------------
CREATE TYPE email_status AS ENUM ('queued', 'sending', 'sent', 'failed', 'cancelled');

CREATE TABLE email_outbox (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    to_address      TEXT NOT NULL,
    to_name         TEXT,
    cc              TEXT[] NOT NULL DEFAULT '{}',
    bcc             TEXT[] NOT NULL DEFAULT '{}',
    subject         TEXT NOT NULL,
    template        TEXT NOT NULL,             -- e.g. user.welcome, password.reset
    payload         JSONB NOT NULL DEFAULT '{}'::jsonb,
    html_body       TEXT,
    text_body       TEXT,
    status          email_status NOT NULL DEFAULT 'queued',
    attempts        INTEGER NOT NULL DEFAULT 0,
    last_error      TEXT,
    provider        TEXT,                      -- snapshot of which driver sent it
    provider_msg_id TEXT,
    scheduled_for   TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_email_outbox_status_sched ON email_outbox(status, scheduled_for);
CREATE INDEX idx_email_outbox_created      ON email_outbox(created_at DESC);
CREATE TRIGGER set_updated_at_email_outbox BEFORE UPDATE ON email_outbox
    FOR EACH ROW EXECUTE FUNCTION trg_set_updated_at();

-- ---------------------------------------------------------------------
-- PASSWORD RESETS
-- ---------------------------------------------------------------------
CREATE TABLE password_resets (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash   TEXT NOT NULL UNIQUE,
    requested_ip INET,
    user_agent   TEXT,
    expires_at   TIMESTAMPTZ NOT NULL,
    used_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_password_resets_user ON password_resets(user_id);
CREATE INDEX idx_password_resets_exp  ON password_resets(expires_at);

-- ---------------------------------------------------------------------
-- IN-APP NOTIFICATIONS
-- ---------------------------------------------------------------------
CREATE TABLE notifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind       TEXT NOT NULL,                  -- e.g. case.published, account.approved
    title      TEXT NOT NULL,
    body       TEXT,
    link       TEXT,
    metadata   JSONB NOT NULL DEFAULT '{}'::jsonb,
    read_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_notifications_user_unread ON notifications(user_id, read_at, created_at DESC);

COMMIT;
