-- 000001_create_party_registry.up.sql
CREATE TABLE IF NOT EXISTS party_registry (
    bic                  VARCHAR(11)  NOT NULL,
    role                 VARCHAR(10)  NOT NULL,
    country_of_residence CHAR(2)      NOT NULL,
    name                 VARCHAR(255) NOT NULL,

    CONSTRAINT pk_party_registry PRIMARY KEY (bic),
    CONSTRAINT chk_party_role    CHECK (role IN ('debtor', 'creditor'))
);