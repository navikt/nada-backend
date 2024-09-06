-- +goose Up
DROP VIEW dataset_view;

CREATE VIEW dataset_view AS(
    SELECT
        ds.id as ds_id,
        ds.name as ds_name,
        ds.description as ds_description,
        ds.created as ds_created,
        ds.last_modified as ds_last_modified,
        ds.slug as ds_slug,
        ds.pii as pii,
        ds.keywords as ds_keywords,
        ds.repo as ds_repo,
        dsrc.id AS bq_id,
        dsrc.created as bq_created,
        dsrc.last_modified as bq_last_modified,
        dsrc.expires as bq_expires,
        dsrc.description as bq_description,
        dsrc.missing_since as bq_missing_since,
        dsrc.pii_tags as pii_tags,
        dsrc.project_id as bq_project,
        dsrc.dataset as bq_dataset,
        dsrc.table_name as bq_table_name,
        dsrc.table_type as bq_table_type,
        dsrc.pseudo_columns as pseudo_columns,
        dsrc.schema as bq_schema,
        ds.dataproduct_id as ds_dp_id,
        dm.services as mapping_services,
        da.id as access_id,
        da.subject as access_subject,
        da.owner as access_owner,
        da.granter as access_granter,
        da.expires as access_expires,
        da.created as access_created,
        da.revoked as access_revoked,
        da.access_request_id as access_request_id,
        mm.database_id as mb_database_id,
        mm.deleted_at as mb_deleted_at
    FROM
        datasets ds
        LEFT JOIN (
            SELECT
                *
            FROM
                datasource_bigquery
            WHERE
                is_reference = false
        ) dsrc ON ds.id = dsrc.dataset_id
        LEFT JOIN third_party_mappings dm ON ds.id = dm.dataset_id
        LEFT JOIN dataset_access da ON ds.id = da.dataset_id
        LEFT JOIN metabase_metadata mm ON ds.id = mm.dataset_id
);

-- +goose Down
DROP VIEW dataset_view;

CREATE VIEW dataset_view AS(
    SELECT
        ds.id as ds_id,
        ds.name as ds_name,
        ds.description as ds_description,
        ds.created as ds_created,
        ds.last_modified as ds_last_modified,
        ds.slug as ds_slug,
        ds.pii as pii,
        ds.keywords as ds_keywords,
        dsrc.id AS bq_id,
        dsrc.created as bq_created,
        dsrc.last_modified as bq_last_modified,
        dsrc.expires as bq_expires,
        dsrc.description as bq_description,
        dsrc.missing_since as bq_missing_since,
        dsrc.pii_tags as pii_tags,
        dsrc.project_id as bq_project,
        dsrc.dataset as bq_dataset,
        dsrc.table_name as bq_table_name,
        dsrc.table_type as bq_table_type,
        dsrc.pseudo_columns as pseudo_columns,
        dsrc.schema as bq_schema,
        ds.dataproduct_id as ds_dp_id,
        dm.services as mapping_services,
        da.id as access_id,
        da.subject as access_subject,
        da.owner as access_owner,
        da.granter as access_granter,
        da.expires as access_expires,
        da.created as access_created,
        da.revoked as access_revoked,
        da.access_request_id as access_request_id,
        mm.database_id as mb_database_id,
        mm.deleted_at as mb_deleted_at
    FROM
        datasets ds
        LEFT JOIN (
            SELECT
                *
            FROM
                datasource_bigquery
            WHERE
                is_reference = false
        ) dsrc ON ds.id = dsrc.dataset_id
        LEFT JOIN third_party_mappings dm ON ds.id = dm.dataset_id
        LEFT JOIN dataset_access da ON ds.id = da.dataset_id
        LEFT JOIN metabase_metadata mm ON ds.id = mm.dataset_id
);
