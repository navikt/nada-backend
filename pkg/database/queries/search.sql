-- name: Search :many
SELECT element_id::uuid,
       element_type::text,
       ts_rank_cd(tsv_document, query)                                                                                               AS rank,
       ts_headline('norwegian', "description", query,
                   'MinWords=10, MaxWords=20, MaxFragments=2 FragmentDelimiter=" … " StartSel="((START))" StopSel="((STOP))"')::text AS excerpt
FROM search,
     websearch_to_tsquery('norwegian', @query) query
WHERE (
    CASE
        WHEN array_length(@types::text[], 1) > 0 THEN "element_type" = ANY (@types)
        ELSE TRUE
        END
    )
  AND (
    CASE
        WHEN array_length(@keyword::text[], 1) > 0 THEN "keywords" && @keyword
        ELSE TRUE
        END
    )
  AND (
    CASE
        WHEN @query :: text != '' THEN "tsv_document" @@ query
        ELSE TRUE
        END
    )
  AND (
    CASE
        WHEN array_length(@grp::text[], 1) > 0 THEN "group" = ANY (@grp)
        ELSE TRUE
        END
    )
  AND (
    CASE
        WHEN array_length(@service::text[], 1) > 0 THEN "services" && @service
        ELSE TRUE
        END
    )
ORDER BY rank DESC, created ASC
LIMIT @lim OFFSET @offs;

-- name: SimpleSearch :many
SELECT *
FROM (SELECT "dataproducts"."id"                        AS "element_id",
             'dataproduct'::text                        AS "element_type",
             coalesce("dataproducts"."description", '') AS "description",
             "dataproducts"."name",
             "dataproducts"."keywords",
             "dataproducts"."group",
             "dataproducts"."created",
             "dataproducts"."last_modified",
             "tpm"."services"
      FROM "dataproducts"
               LEFT JOIN "third_party_mappings" "tpm" ON "tpm"."dataproduct_id" = "dataproducts"."id"
      UNION
      SELECT "stories"."id"                        AS "element_id",
             'story'::text                         AS "element_type",
             coalesce("stories"."description", '') AS "description",
             "stories"."name",
             '{}'                                  AS "keywords",
             "stories"."group",
             "stories"."created",
             "stories"."last_modified",
             '{}'                                  AS "services"
      FROM (SELECT "id",
                   "name",
                   "group",
                   "created",
                   "last_modified",
                   "keywords",
                   (SELECT string_agg("spec" ->> 'content', ' ')
                    FROM (SELECT "spec"
                          FROM "story_views"
                          WHERE "story_id" = "stories"."id"
                            AND "type" IN ('markdown', 'header')
                          ORDER BY "sort" ASC) "views") AS "description"
            FROM "stories") "stories") AS data
WHERE similarity(@query, name) > 0.05
   OR similarity(@query, description) > 0.05
    AND (
          CASE
              WHEN array_length(@types::text[], 1) > 0 THEN "element_type" = ANY (@types)
              ELSE TRUE
              END
          )
    AND (
          CASE
              WHEN array_length(@keyword::text[], 1) > 0 THEN "keywords" && @keyword
              ELSE TRUE
              END
          )
    AND (
          CASE
              WHEN array_length(@grp::text[], 1) > 0 THEN "group" = ANY (@grp)
              ELSE TRUE
              END
          )
    AND (
          CASE
              WHEN array_length(@service::text[], 1) > 0 THEN "services" && @service
              ELSE TRUE
              END
          )
ORDER BY GREATEST(similarity(@query, name), similarity(@query, description)) DESC
LIMIT @lim OFFSET @offs;