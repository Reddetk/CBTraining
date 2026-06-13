-- 000005_create_outbox.up.sql
CREATE TABLE IF NOT EXISTS outbox (
    txid       VARCHAR(35)  NOT NULL,
    topic      VARCHAR(255) NOT NULL,
    payload    JSONB        NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL,
    sent_at    TIMESTAMPTZ  NULL,

    CONSTRAINT pk_outbox PRIMARY KEY (txid)
);

-- OutboxRelay read only unsended 
CREATE INDEX IF NOT EXISTS idx_outbox_unsent
    ON outbox (created_at ASC)
    WHERE sent_at IS NULL;