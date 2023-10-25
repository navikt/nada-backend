WITH
-- Temporary table with all dataproducts
dataproducts AS (
    SELECT * FROM EXTERNAL_QUERY(
    'europe-north1.bq-nada-backend',
    '''

    -- Lag en variabel for varsjonering
    WITH constants (version) as (
        values (now())
    )

    SELECT
        id::text,
        name,
        "group" AS owner,
        pii,
        created,
        last_modified,
        "type"::text,
        version
    FROM dataproducts,constants
    ''')
),
-- All active access grouped by dataproduct
access AS (
    SELECT * FROM EXTERNAL_QUERY(
    'europe-north1.bq-nada-backend',
    '''
    SELECT dataproduct_id::text, count(1) AS count
    FROM dataproduct_access
    WHERE revoked IS NULL AND (expires IS NULL OR expires < now())
    GROUP BY dataproduct_id
    ''')
),
-- Number of requesters grouped by dataproduct
requesters AS (
    SELECT * FROM EXTERNAL_QUERY(
    'europe-north1.bq-nada-backend',
    '''
    SELECT dataproduct_id::text, count(1) AS count
    FROM dataproduct_requesters
    GROUP BY dataproduct_id
    ''')
),
-- Dataproducts with active access to 'all-users@nav.no'
all_users AS (
    SELECT * FROM EXTERNAL_QUERY(
    'europe-north1.bq-nada-backend',
    '''
    SELECT dataproduct_id::text, true AS nav_internal
    FROM dataproduct_access
    WHERE revoked IS NULL AND (expires IS NULL OR expires < now()) AND subject = 'group:all-users@nav.no'
    ''')
)

SELECT
    dataproducts.*,
    COALESCE(all_users.nav_internal,false) AS nav_internal,
    COALESCE(requesters.count,0) AS requesters_count,
    COALESCE(access.count,0) AS access_count
FROM dataproducts
LEFT JOIN requesters ON requesters.dataproduct_id = dataproducts.id
LEFT JOIN access ON access.dataproduct_id = dataproducts.id
LEFT JOIN all_users ON all_users.dataproduct_id = dataproducts.id
;
