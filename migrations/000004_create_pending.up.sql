-- 000004_create_pending.up.sql
CREATE TABLE IF NOT EXISTS pending (
    tx_id      VARCHAR(35)  NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL,

    CONSTRAINT pk_pending    PRIMARY KEY (tx_id),
    CONSTRAINT fk_pending_tx FOREIGN KEY (tx_id)
        REFERENCES tx_history (id) ON DELETE CASCADE ON UPDATE CASCADE
);

-- RestorePending ORDER BY created_at ASC - main req to this table
CREATE INDEX IF NOT EXISTS idx_pending_fifo ON pending (created_at ASC);