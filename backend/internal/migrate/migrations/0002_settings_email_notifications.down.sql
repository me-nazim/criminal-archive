BEGIN;

DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS password_resets;
DROP TRIGGER IF EXISTS set_updated_at_email_outbox ON email_outbox;
DROP TABLE IF EXISTS email_outbox;
DROP TYPE IF EXISTS email_status;
DROP TABLE IF EXISTS app_settings;

COMMIT;
