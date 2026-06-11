-- 000005_create_outbox.down.sql
DROP INDEX IF EXISTS idx_outbox_unsent;
DROP TABLE IF EXISTS outbox;