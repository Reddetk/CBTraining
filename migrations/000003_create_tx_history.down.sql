-- 000003_create_tx_history.down.sql
DROP INDEX IF EXISTS idx_tx_history_creditor;
DROP INDEX IF EXISTS idx_tx_history_debtor;
DROP INDEX IF EXISTS idx_tx_history_status_updated;
DROP TABLE IF EXISTS tx_history;