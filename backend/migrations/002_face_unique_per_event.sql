-- FaceIds are unique per event collection, not globally (provider-agnostic).
ALTER TABLE photo_faces DROP CONSTRAINT IF EXISTS photo_faces_aws_face_id_key;
ALTER TABLE photo_faces ADD CONSTRAINT photo_faces_event_face_key UNIQUE(event_id,aws_face_id);
