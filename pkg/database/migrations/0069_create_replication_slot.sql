-- +goose Up
SELECT PG_CREATE_LOGICAL_REPLICATION_SLOT('ds_replication', 'pgoutput');

-- +goose Down
SELECT pg_drop_replication_slot('ds_replication');
