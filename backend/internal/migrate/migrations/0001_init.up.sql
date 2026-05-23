-- =====================================================================
-- Tansiq Information Portal — Initial schema
-- PostgreSQL 18
-- =====================================================================

BEGIN;

-- Extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;     -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS citext;        -- case-insensitive emails
CREATE EXTENSION IF NOT EXISTS pg_trgm;       -- fuzzy / full-text search support

-- ---------------------------------------------------------------------
-- ENUMS
-- ---------------------------------------------------------------------
CREATE TYPE user_role AS ENUM (
    'super_admin',
    'admin',
    'moderator',
    'contributor',
    'viewer'
);

CREATE TYPE user_status AS ENUM (
    'pending',
    'approved',
    'suspended',
    'rejected'
);

CREATE TYPE submission_status AS ENUM (
    'draft',
    'pending_review',
    'in_verification',
    'approved',
    'published',
    'rejected',
    'archived'
);

CREATE TYPE person_type AS ENUM (
    'victim',
    'accused',
    'witness',
    'other'
);

CREATE TYPE attachment_kind AS ENUM (
    'public',     -- visible to everyone
    'hidden',     -- hidden from public; visible to admins only
    'internal'    -- internal admin notes / reference docs
);

CREATE TYPE verification_status AS ENUM (
    'unassigned',
    'assigned',
    'in_progress',
    'verified',
    'rejected'
);

-- ---------------------------------------------------------------------
-- USERS
-- ---------------------------------------------------------------------
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           CITEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    full_name       TEXT NOT NULL,
    display_name    TEXT,
    role            user_role NOT NULL DEFAULT 'contributor',
    status          user_status NOT NULL DEFAULT 'pending',
    phone           TEXT,
    avatar_url      TEXT,
    bio             TEXT,
    last_login_at   TIMESTAMPTZ,
    approved_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_users_role_status ON users(role, status);

-- ---------------------------------------------------------------------
-- LOCATIONS  (Bangladesh: Country -> Division -> District -> Upazila;
--             Other countries: Country only, free-text address)
-- ---------------------------------------------------------------------
CREATE TABLE countries (
    id          SERIAL PRIMARY KEY,
    iso2        CHAR(2) NOT NULL UNIQUE,
    iso3        CHAR(3) NOT NULL UNIQUE,
    name_en     TEXT NOT NULL,
    name_bn     TEXT,
    phone_code  TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE divisions (
    id          SERIAL PRIMARY KEY,
    country_id  INTEGER NOT NULL REFERENCES countries(id) ON DELETE CASCADE,
    name_en     TEXT NOT NULL,
    name_bn     TEXT,
    bbs_code    TEXT,
    UNIQUE (country_id, name_en)
);

CREATE TABLE districts (
    id           SERIAL PRIMARY KEY,
    division_id  INTEGER NOT NULL REFERENCES divisions(id) ON DELETE CASCADE,
    name_en      TEXT NOT NULL,
    name_bn      TEXT,
    bbs_code     TEXT,
    UNIQUE (division_id, name_en)
);

CREATE TABLE upazilas (
    id           SERIAL PRIMARY KEY,
    district_id  INTEGER NOT NULL REFERENCES districts(id) ON DELETE CASCADE,
    name_en      TEXT NOT NULL,
    name_bn      TEXT,
    bbs_code     TEXT,
    UNIQUE (district_id, name_en)
);
CREATE INDEX idx_divisions_country ON divisions(country_id);
CREATE INDEX idx_districts_division ON districts(division_id);
CREATE INDEX idx_upazilas_district ON upazilas(district_id);

-- ---------------------------------------------------------------------
-- CRIME TYPES
-- ---------------------------------------------------------------------
CREATE TABLE crime_types (
    id          SERIAL PRIMARY KEY,
    slug        TEXT NOT NULL UNIQUE,
    name_en     TEXT NOT NULL,
    name_bn     TEXT NOT NULL,
    description TEXT,
    severity    SMALLINT NOT NULL DEFAULT 1,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE
);

-- ---------------------------------------------------------------------
-- PERSONS  (victims & accused share this table; one person can be linked
--           to many cases via case_persons)
-- ---------------------------------------------------------------------
CREATE TABLE persons (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug               TEXT NOT NULL UNIQUE,
    full_name_bn       TEXT,
    full_name_en       TEXT,
    aliases            TEXT[] NOT NULL DEFAULT '{}',
    primary_type       person_type NOT NULL,
    gender             TEXT,
    date_of_birth      DATE,
    photo_url          TEXT,

    occupation         TEXT,
    organization       TEXT,
    designation        TEXT,

    country_id         INTEGER REFERENCES countries(id) ON DELETE SET NULL,
    division_id        INTEGER REFERENCES divisions(id) ON DELETE SET NULL,
    district_id        INTEGER REFERENCES districts(id) ON DELETE SET NULL,
    upazila_id         INTEGER REFERENCES upazilas(id)  ON DELETE SET NULL,
    address_line       TEXT,

    public_bio_bn      TEXT,
    public_bio_en      TEXT,
    internal_notes     TEXT,                 -- admin-only

    is_anonymous       BOOLEAN NOT NULL DEFAULT FALSE,  -- mostly for victims
    nid_hash           TEXT,                 -- hashed NID; never store raw

    status             submission_status NOT NULL DEFAULT 'pending_review',
    submitted_by       UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_by        UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_at        TIMESTAMPTZ,
    published_at       TIMESTAMPTZ,

    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_persons_status         ON persons(status);
CREATE INDEX idx_persons_country        ON persons(country_id);
CREATE INDEX idx_persons_district       ON persons(district_id);
CREATE INDEX idx_persons_primary_type   ON persons(primary_type);
CREATE INDEX idx_persons_name_en_trgm   ON persons USING gin (full_name_en gin_trgm_ops);
CREATE INDEX idx_persons_name_bn_trgm   ON persons USING gin (full_name_bn gin_trgm_ops);

-- ---------------------------------------------------------------------
-- CASES  (one incident)
-- ---------------------------------------------------------------------
CREATE TABLE cases (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_number        TEXT NOT NULL UNIQUE,        -- e.g. TIP-2026-00045
    slug               TEXT NOT NULL UNIQUE,
    title_bn           TEXT NOT NULL,
    title_en           TEXT,
    summary_bn         TEXT,
    summary_en         TEXT,
    description_bn     TEXT,
    description_en     TEXT,
    internal_notes     TEXT,                          -- admin-only

    incident_date      DATE,
    incident_time      TIME,

    country_id         INTEGER REFERENCES countries(id) ON DELETE SET NULL,
    division_id        INTEGER REFERENCES divisions(id) ON DELETE SET NULL,
    district_id        INTEGER REFERENCES districts(id) ON DELETE SET NULL,
    upazila_id         INTEGER REFERENCES upazilas(id)  ON DELETE SET NULL,
    location_text      TEXT,                          -- free-text / international

    crime_type_id      INTEGER REFERENCES crime_types(id) ON DELETE SET NULL,
    case_status        TEXT,                          -- under_investigation / court / closed / etc
    severity           SMALLINT,

    cover_image_url    TEXT,
    tags               TEXT[] NOT NULL DEFAULT '{}',

    status             submission_status NOT NULL DEFAULT 'pending_review',
    submitted_by       UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_by        UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_at        TIMESTAMPTZ,
    published_at       TIMESTAMPTZ,

    view_count         BIGINT NOT NULL DEFAULT 0,
    download_count     BIGINT NOT NULL DEFAULT 0,

    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_cases_status            ON cases(status);
CREATE INDEX idx_cases_published_at      ON cases(published_at DESC);
CREATE INDEX idx_cases_incident_date     ON cases(incident_date DESC);
CREATE INDEX idx_cases_country           ON cases(country_id);
CREATE INDEX idx_cases_district          ON cases(district_id);
CREATE INDEX idx_cases_crime_type        ON cases(crime_type_id);
CREATE INDEX idx_cases_tags_gin          ON cases USING gin (tags);
CREATE INDEX idx_cases_title_bn_trgm     ON cases USING gin (title_bn gin_trgm_ops);
CREATE INDEX idx_cases_title_en_trgm     ON cases USING gin (title_en gin_trgm_ops);

-- A counter to drive case_number generation (TIP-{YYYY}-{seq:05d})
CREATE TABLE case_number_counters (
    year    INTEGER PRIMARY KEY,
    seq     INTEGER NOT NULL DEFAULT 0
);

-- ---------------------------------------------------------------------
-- CASE <-> PERSONS (many-to-many)
-- ---------------------------------------------------------------------
CREATE TABLE case_persons (
    case_id     UUID NOT NULL REFERENCES cases(id)   ON DELETE CASCADE,
    person_id   UUID NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    role        person_type NOT NULL,
    notes       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (case_id, person_id, role)
);
CREATE INDEX idx_case_persons_person ON case_persons(person_id);
CREATE INDEX idx_case_persons_case   ON case_persons(case_id);

-- ---------------------------------------------------------------------
-- CASE ATTACHMENTS  (files in R2)
-- File naming convention:
--   TIP-{YEAR}-{SEQ:05d}_{kind}_{idx:02d}.{ext}
-- ---------------------------------------------------------------------
CREATE TABLE case_attachments (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id            UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    kind               attachment_kind NOT NULL DEFAULT 'public',
    sequence_no        INTEGER NOT NULL,
    original_filename  TEXT NOT NULL,
    stored_filename    TEXT NOT NULL,                  -- e.g. TIP-2026-00045_evidence_03.jpg
    storage_key        TEXT NOT NULL,                  -- full R2 object key
    public_url         TEXT,
    mime_type          TEXT NOT NULL,
    size_bytes         BIGINT NOT NULL,
    checksum_sha256    TEXT,
    width              INTEGER,
    height             INTEGER,
    duration_seconds   INTEGER,
    caption_bn         TEXT,
    caption_en         TEXT,
    uploaded_by        UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (case_id, kind, sequence_no)
);
CREATE INDEX idx_attachments_case ON case_attachments(case_id);
CREATE INDEX idx_attachments_kind ON case_attachments(kind);

-- ---------------------------------------------------------------------
-- CASE TIMELINE (events / progression / updates)
-- ---------------------------------------------------------------------
CREATE TABLE case_timeline (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id         UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    event_date      DATE NOT NULL,
    event_time      TIME,
    title_bn        TEXT NOT NULL,
    title_en        TEXT,
    description_bn  TEXT,
    description_en  TEXT,
    source_url      TEXT,
    is_internal     BOOLEAN NOT NULL DEFAULT FALSE,
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_timeline_case_date ON case_timeline(case_id, event_date DESC);

-- ---------------------------------------------------------------------
-- NEWS SOURCES (external links cited by a case)
-- ---------------------------------------------------------------------
CREATE TABLE news_sources (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id       UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    url           TEXT NOT NULL,
    title         TEXT,
    source_name   TEXT,
    published_at  TIMESTAMPTZ,
    archived_url  TEXT,                                  -- e.g. archive.org snapshot
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_news_case ON news_sources(case_id);

-- ---------------------------------------------------------------------
-- VERIFICATION ASSIGNMENTS
-- ---------------------------------------------------------------------
CREATE TABLE verification_assignments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id       UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    assigned_to   UUID REFERENCES users(id) ON DELETE SET NULL,
    assigned_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    status        verification_status NOT NULL DEFAULT 'unassigned',
    notes         TEXT,
    assigned_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at  TIMESTAMPTZ
);
CREATE INDEX idx_verif_case   ON verification_assignments(case_id);
CREATE INDEX idx_verif_status ON verification_assignments(status);

-- ---------------------------------------------------------------------
-- AUDIT LOGS
-- ---------------------------------------------------------------------
CREATE TABLE audit_logs (
    id           BIGSERIAL PRIMARY KEY,
    user_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    action       TEXT NOT NULL,
    target_type  TEXT,
    target_id    TEXT,
    metadata     JSONB,
    ip_address   INET,
    user_agent   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_user        ON audit_logs(user_id);
CREATE INDEX idx_audit_created     ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_target      ON audit_logs(target_type, target_id);

-- ---------------------------------------------------------------------
-- SESSIONS (refresh tokens — opaque)
-- ---------------------------------------------------------------------
CREATE TABLE sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_hash    TEXT NOT NULL,
    user_agent      TEXT,
    ip_address      INET,
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_sessions_user ON sessions(user_id);

-- ---------------------------------------------------------------------
-- updated_at trigger
-- ---------------------------------------------------------------------
CREATE OR REPLACE FUNCTION trg_set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_updated_at_users    BEFORE UPDATE ON users    FOR EACH ROW EXECUTE FUNCTION trg_set_updated_at();
CREATE TRIGGER set_updated_at_persons  BEFORE UPDATE ON persons  FOR EACH ROW EXECUTE FUNCTION trg_set_updated_at();
CREATE TRIGGER set_updated_at_cases    BEFORE UPDATE ON cases    FOR EACH ROW EXECUTE FUNCTION trg_set_updated_at();

COMMIT;
