-- 000004_create_pending.down.sql
DROP INDEX IF EXISTS idx_pending_fifo;
DROP TABLE IF EXISTS pending;