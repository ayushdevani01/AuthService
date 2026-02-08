-- 000001_init_schema.down.sql
-- Rollback initial schema

DROP TRIGGER IF EXISTS update_apps_updated_at ON apps;
DROP TRIGGER IF EXISTS update_developers_updated_at ON developers;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS activity_logs;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS user_identities;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS oauth_providers;
DROP TABLE IF EXISTS signing_keys;
DROP TABLE IF EXISTS apps;
DROP TABLE IF EXISTS developers;
