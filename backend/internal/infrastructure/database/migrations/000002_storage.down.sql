-- Storage Context Rollback
-- Drop tables in reverse order of creation

DROP TABLE IF EXISTS upload_parts;
DROP TABLE IF EXISTS upload_sessions;
DROP TABLE IF EXISTS archived_file_versions;
DROP TABLE IF EXISTS archived_files;
DROP TABLE IF EXISTS file_versions;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS folder_paths;
DROP TABLE IF EXISTS folders;

DROP TYPE IF EXISTS upload_session_status;
DROP TYPE IF EXISTS file_status;
DROP TYPE IF EXISTS owner_type;
