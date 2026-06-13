-- 000002_create_pa_registry.up.sql
CREATE TABLE IF NOT EXISTS pa_registry (
    iban             VARCHAR(34) NOT NULL,
    account_currency CHAR(3)     NOT NULL,
    party_bic         VARCHAR(11) NOT NULL,

    CONSTRAINT pk_pa_registry PRIMARY KEY (iban),
    CONSTRAINT fk_pa_party    FOREIGN KEY (party_bic)
        REFERENCES party_registry (bic) ON DELETE RESTRICT ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_pa_registry_party_bic ON pa_registry (party_bic);