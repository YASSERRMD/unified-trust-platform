DROP RULE IF EXISTS no_delete_audit ON audit_events;
DROP RULE IF EXISTS no_update_audit ON audit_events;
DROP TABLE IF EXISTS audit_events;
