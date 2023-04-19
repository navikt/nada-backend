-- +goose Up
-- +goose StatementBegin
DO
$$
    BEGIN
        IF EXISTS(SELECT * FROM pg_roles WHERE rolname = 'datastream') THEN
            ALTER USER "nada-backend" WITH REPLICATION;
            CREATE PUBLICATION "ds_publication" FOR ALL TABLES;
            
            ALTER DEFAULT PRIVILEGES IN SCHEMA PUBLIC GRANT SELECT ON TABLES TO "datastream";
            GRANT SELECT ON ALL TABLES IN SCHEMA PUBLIC TO "datastream";
            ALTER USER "datastream" WITH REPLICATION;
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
            ALTER USER "datastream" WITH NOREPLICATION;
            REVOKE SELECT ON ALL TABLES IN SCHEMA PUBLIC FROM "datastream";
            ALTER DEFAULT PRIVILEGES IN SCHEMA PUBLIC REVOKE SELECT ON TABLES FROM "datastream";
            
            DROP PUBLICATION "ds_publication";
            ALTER USER "nada-backend" WITH NOREPLICATION;
        END IF;
    END
$$ LANGUAGE 'plpgsql';
-- +goose StatementEnd