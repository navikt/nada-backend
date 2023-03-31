-- +goose Up
-- +goose StatementBegin
DO
$$
    BEGIN
        IF EXISTS(SELECT * FROM pg_roles WHERE rolname = 'datastream') THEN
            ALTER DEFAULT PRIVILEGES IN SCHEMA PUBLIC GRANT SELECT ON TABLES TO "datastream";
            GRANT SELECT ON ALL TABLES IN SCHEMA PUBLIC TO "datastream";

            ALTER USER "nada-backend" WITH REPLICATION;
            ALTER USER "datastream" WITH REPLICATION;
            CREATE PUBLICATION "ds_publication" FOR ALL TABLES;
            SELECT PG_CREATE_LOGICAL_REPLICATION_SLOT('ds_replication', 'pgoutput');
        END IF;
    END
$$ LANGUAGE 'plpgsql';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO
$$
    BEGIN
        IF EXISTS(SELECT * FROM pg_roles WHERE rolname = 'datastream') THEN
            SELECT pg_drop_replication_slot('ds_replication');
            DROP PUBLICATION "ds_publication";
            ALTER USER "datastream" WITH NOREPLICATION;
            ALTER USER "nada-backend" WITH NOREPLICATION;

            REVOKE SELECT ON ALL TABLES IN SCHEMA PUBLIC FROM "datastream";
            ALTER DEFAULT PRIVILEGES IN SCHEMA PUBLIC REVOKE SELECT ON TABLES FROM "datastream";
        END IF;
    END
$$ LANGUAGE 'plpgsql';
-- +goose StatementEnd
