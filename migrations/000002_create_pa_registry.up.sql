-- 000002_create_pa_registry.up.sql
CREATE TABLE IF NOT EXISTS pa_registry (
    id               VARCHAR(34) NOT NULL,
    account_currency CHAR(3)     NOT NULL,
    party_id         VARCHAR(11) NOT NULL,

    CONSTRAINT pk_pa_registry PRIMARY KEY (id),
    CONSTRAINT fk_pa_party    FOREIGN KEY (party_id)
        REFERENCES party_registry (id) ON DELETE RESTRICT ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_pa_registry_party_id ON pa_registry (party_id);