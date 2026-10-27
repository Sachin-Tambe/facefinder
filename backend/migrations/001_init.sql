-- 001_init: core multi-tenant schema. Metadata only; never photo bytes.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, name TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now());

CREATE TABLE organizations(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name TEXT NOT NULL, slug TEXT NOT NULL UNIQUE,
  plan TEXT NOT NULL DEFAULT 'starter',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now());

CREATE TABLE organization_members(
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role TEXT NOT NULL CHECK(role IN ('organization_owner','admin','photographer','staff','viewer')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(), PRIMARY KEY(organization_id,user_id));

CREATE TABLE events(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '',
  event_code TEXT NOT NULL, event_token_hash TEXT NOT NULL,
  rekognition_collection_id TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','paused','expired','purging','purged')),
  starts_at TIMESTAMPTZ, ends_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ, face_data_expires_at TIMESTAMPTZ, face_collection_deleted_at TIMESTAMPTZ,
  photos_expire_at TIMESTAMPTZ, photos_deleted_at TIMESTAMPTZ,
  gallery_ttl_hours INT NOT NULL DEFAULT 24, whatsapp_session_ttl_hours INT NOT NULL DEFAULT 24,
  selfie_retention_minutes INT NOT NULL DEFAULT 60,
  auto_delete_enabled BOOLEAN NOT NULL DEFAULT true, downloads_enabled BOOLEAN NOT NULL DEFAULT true,
  created_by UUID REFERENCES users(id), created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(organization_id,event_code));

CREATE TABLE photos(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  r2_original_key TEXT NOT NULL, r2_web_key TEXT, r2_thumb_key TEXT, r2_analysis_key TEXT,
  file_name TEXT NOT NULL, mime_type TEXT NOT NULL, file_size BIGINT NOT NULL DEFAULT 0,
  width INT NOT NULL DEFAULT 0, height INT NOT NULL DEFAULT 0,
  processing_status TEXT NOT NULL DEFAULT 'uploaded' CHECK(processing_status IN ('uploaded','queued','processing','ready','no_face','failed')),
  face_count INT NOT NULL DEFAULT 0, error TEXT,
  uploaded_at TIMESTAMPTZ NOT NULL DEFAULT now(), processed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now());
CREATE INDEX photos_event_status ON photos(event_id,processing_status);

CREATE TABLE photo_faces(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  photo_id UUID NOT NULL REFERENCES photos(id) ON DELETE CASCADE,
  aws_face_id TEXT NOT NULL, aws_image_id TEXT NOT NULL,
  bounding_box_json JSONB NOT NULL DEFAULT '{}', detection_confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(event_id,aws_face_id));
CREATE INDEX photo_faces_photo ON photo_faces(photo_id);

CREATE TABLE whatsapp_sessions(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  phone_e164 TEXT NOT NULL, gateway_number TEXT NOT NULL DEFAULT '',
  event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(), last_activity_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL, status TEXT NOT NULL DEFAULT 'active',
  UNIQUE(phone_e164,gateway_number));
CREATE INDEX wa_sessions_phone ON whatsapp_sessions(phone_e164,status,expires_at);

CREATE TABLE search_requests(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  phone_e164 TEXT NOT NULL, selfie_r2_key TEXT, status TEXT NOT NULL DEFAULT 'received',
  detected_face_count INT, error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(), completed_at TIMESTAMPTZ, deleted_at TIMESTAMPTZ);
CREATE INDEX searches_event ON search_requests(event_id,created_at DESC);

CREATE TABLE search_results(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  search_id UUID NOT NULL REFERENCES search_requests(id) ON DELETE CASCADE,
  photo_id UUID NOT NULL REFERENCES photos(id) ON DELETE CASCADE,
  best_face_id TEXT NOT NULL, similarity DOUBLE PRECISION NOT NULL,
  result_type TEXT NOT NULL CHECK(result_type IN ('strong','possible')), rank INT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(search_id,photo_id));

CREATE TABLE gallery_tokens(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  search_id UUID NOT NULL REFERENCES search_requests(id) ON DELETE CASCADE,
  event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  phone_e164 TEXT NOT NULL, token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL, revoked_at TIMESTAMPTZ, last_accessed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now());

CREATE TABLE processing_jobs(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id UUID REFERENCES events(id) ON DELETE CASCADE, photo_id UUID REFERENCES photos(id) ON DELETE CASCADE,
  kind TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'queued', attempts INT NOT NULL DEFAULT 0,
  last_error TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now());

CREATE TABLE consents(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id UUID REFERENCES events(id) ON DELETE CASCADE, phone_e164 TEXT, kind TEXT NOT NULL,
  granted BOOLEAN NOT NULL DEFAULT true, created_at TIMESTAMPTZ NOT NULL DEFAULT now());

CREATE TABLE audit_logs(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID, event_id UUID, user_id UUID, actor TEXT NOT NULL DEFAULT 'system',
  action TEXT NOT NULL, details JSONB NOT NULL DEFAULT '{}', request_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now());
CREATE INDEX audit_event ON audit_logs(event_id,created_at DESC);

-- Row-level isolation backstop (app also filters organization_id on every query).
ALTER TABLE events ENABLE ROW LEVEL SECURITY;
ALTER TABLE photos ENABLE ROW LEVEL SECURITY;
CREATE POLICY org_isolation_events ON events USING (organization_id = NULLIF(current_setting('app.org_id',true),'')::uuid);
CREATE POLICY org_isolation_photos ON photos USING (organization_id = NULLIF(current_setting('app.org_id',true),'')::uuid);
