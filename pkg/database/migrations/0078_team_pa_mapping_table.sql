-- +goose Up
CREATE TABLE "team_productarea_mapping" (
    "team_id"         TEXT NOT NULL,
    "product_area_id" TEXT,
    PRIMARY KEY(team_id)
);

WITH combined_team_pa AS (
    SELECT DISTINCT(team_id), product_area_id FROM dataproducts WHERE team_id IS NOT NULL
    UNION
    SELECT DISTINCT(team_id), product_area_id FROM quarto_stories WHERE team_id IS NOT NULL
    UNION
    SELECT DISTINCT(team_id), product_area_id FROM stories WHERE team_id IS NOT NULL
)
INSERT INTO team_productarea_mapping (team_id, product_area_id)
(SELECT team_id, product_area_id FROM combined_team_pa);

ALTER TABLE dataproducts DROP COLUMN product_area_id;
ALTER TABLE quarto_stories DROP COLUMN product_area_id;
ALTER TABLE stories DROP COLUMN product_area_id;

-- +goose Down
ALTER TABLE dataproducts ADD COLUMN product_area_id TEXT;
ALTER TABLE quarto_stories ADD COLUMN product_area_id TEXT;
ALTER TABLE stories ADD COLUMN product_area_id TEXT;

UPDATE dataproducts dp 
SET "product_area_id" = (SELECT product_area_id FROM "team_productarea_mapping" WHERE team_id = dp.team_id);

UPDATE quarto_stories qs 
SET "product_area_id" = (SELECT product_area_id FROM "team_productarea_mapping" WHERE team_id = qs.team_id);

UPDATE stories s
SET "product_area_id" = (SELECT product_area_id FROM "team_productarea_mapping" WHERE team_id = s.team_id);

DROP TABLE "team_productarea_mapping";
