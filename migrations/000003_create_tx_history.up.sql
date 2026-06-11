-- 000003_create_tx_history.up.sql
CREATE TABLE IF NOT EXISTS tx_history (
    id                  VARCHAR(35)    NOT NULL,
    status              VARCHAR(10)    NOT NULL,
    amount              NUMERIC(20, 5) NOT NULL,
    currency            CHAR(3)        NOT NULL,
    end_to_end_id       VARCHAR(35)    NOT NULL,
    transaction_type    CHAR(3)        NOT NULL,
    debtor_account_id   VARCHAR(34)    NOT NULL,
    creditor_account_id VARCHAR(34)    NOT NULL,
    created_at          TIMESTAMPTZ    NOT NULL,
    updated_at          TIMESTAMPTZ    NOT NULL,

    CONSTRAINT pk_tx_history          PRIMARY KEY (id),
    CONSTRAINT uq_tx_end_to_end_id    UNIQUE (end_to_end_id),
    CONSTRAINT chk_tx_status          CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    CONSTRAINT fk_tx_debtor_account   FOREIGN KEY (debtor_account_id)
        REFERENCES pa_registry (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_tx_creditor_account FOREIGN KEY (creditor_account_id)
        REFERENCES pa_registry (id) ON DELETE RESTRICT ON UPDATE CASCADE
);
-- RestorePending: JOIN with pending by tx_id, sorted by pending.created_at — index on the pending side
-- PersistProcessResult: search by PK (id) — automatically covered

-- Monitoring/Dashboards: filter by status
CREATE INDEX IF NOT EXISTS idx_tx_history_status_updated
    ON tx_history (status, updated_at DESC);

-- History of payment by payment account
CREATE INDEX IF NOT EXISTS idx_tx_history_debtor
    ON tx_history (debtor_account_id);

CREATE INDEX IF NOT EXISTS idx_tx_history_creditor
    ON tx_history (creditor_account_id);