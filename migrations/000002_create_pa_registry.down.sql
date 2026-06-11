-- 000002_create_pa_registry.down.sql
DROP INDEX  IF EXISTS idx_pa_registry_party_id;
DROP TABLE  IF EXISTS pa_registry;