-- 000004_create_pending.up.sql
CREATE TABLE IF NOT EXISTS pending (
    txid      VARCHAR(35)  NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL,

    CONSTRAINT pk_pending    PRIMARY KEY (txid),
    CONSTRAINT fk_pending_tx FOREIGN KEY (txid)
        REFERENCES tx_history (txid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- RestorePending ORDER BY created_at ASC - main req to this table
CREATE INDEX IF NOT EXISTS idx_pending_fifo ON pending (created_at ASC);