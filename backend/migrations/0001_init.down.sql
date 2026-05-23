-- Rollback initial schema.
BEGIN;

DROP TRIGGER IF EXISTS set_updated_at_cases   ON cases;
DROP TRIGGER IF EXISTS set_updated_at_persons ON persons;
DROP TRIGGER IF EXISTS set_updated_at_users   ON users;
DROP FUNCTION IF EXISTS trg_set_updated_at();

DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS verification_assignments;
DROP TABLE IF EXISTS news_sources;
DROP TABLE IF EXISTS case_timeline;
DROP TABLE IF EXISTS case_attachments;
DROP TABLE IF EXISTS case_persons;
DROP TABLE IF EXISTS case_number_counters;
DROP TABLE IF EXISTS cases;
DROP TABLE IF EXISTS persons;
DROP TABLE IF EXISTS crime_types;
DROP TABLE IF EXISTS upazilas;
DROP TABLE IF EXISTS districts;
DROP TABLE IF EXISTS divisions;
DROP TABLE IF EXISTS countries;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS verification_status;
DROP TYPE IF EXISTS attachment_kind;
DROP TYPE IF EXISTS person_type;
DROP TYPE IF EXISTS submission_status;
DROP TYPE IF EXISTS user_status;
DROP TYPE IF EXISTS user_role;

COMMIT;
