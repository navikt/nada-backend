-- +goose Up
CREATE TABLE tk_product_areas (
    "id" uuid PRIMARY KEY,
    "name" TEXT,
	"areaType" TEXT
);

CREATE TABLE tk_teams (
    "id" uuid PRIMARY KEY,
    "product_area_id" uuid,
    "name" TEXT,
    CONSTRAINT fk_teams_product_area
        FOREIGN KEY (product_area_id)
            REFERENCES tk_product_areas (id) ON DELETE SET NULL    
);

CREATE VIEW dataproduct_with_teamkatalogen_view AS(
SELECT dp.*, tkt.name as team_name, tkpa.name as pa_name FROM dataproducts dp LEFT JOIN 
	(tk_teams tkt LEFT JOIN tk_product_areas tkpa
	ON tkt.product_area_id = tkpa.id)
	ON dp.team_id = tkt.id::text
);

CREATE VIEW story_with_teamkatalogen_view AS(
SELECT s.*, tkt.name as team_name, tkpa.name as pa_name FROM stories s LEFT JOIN 
	(tk_teams tkt LEFT JOIN tk_product_areas tkpa
	ON tkt.product_area_id = tkpa.id)
	ON s.team_id = tkt.id::text
);

CREATE VIEW insight_product_with_teamkatalogen_view AS(
SELECT isp.*, tkt.name as team_name, tkpa.name as pa_name FROM insight_product isp LEFT JOIN 
	(tk_teams tkt LEFT JOIN tk_product_areas tkpa
	ON tkt.product_area_id = tkpa.id)
	ON isp.team_id = tkt.id::text
);

DROP VIEW IF EXISTS dataproduct_view;

CREATE VIEW dataproduct_view AS(
    SELECT
        dp.id as dp_id,
        dp.name as dp_name,
        dp.description as dp_description,
        dp.group as dp_group,
        dp.created as dp_created,
        dp.last_modified as dp_last_modified,
        dp.slug as dp_slug,
        dp.teamkatalogen_url as teamkatalogen_url,
        dp.team_contact as team_contact,
        dp.team_id as team_id,
		dp.team_name as team_name,
		dp.pa_name as pa_name,
        ds.dataproduct_id as ds_dp_id,
        ds.id as ds_id,
        ds.name as ds_name,
        ds.description as ds_description,
        ds.created as ds_created,
        ds.last_modified as ds_last_modified,
        ds.slug as ds_slug,
        ds.keywords as ds_keywords
    FROM
        dataproduct_with_teamkatalogen_view dp
        LEFT JOIN datasets ds ON dp.id = ds.dataproduct_id
);

-- +goose Down
DROP VIEW dataproduct_view;
DROP VIEW insight_product_with_teamkatalogen_view;
DROP VIEW story_with_teamkatalogen_view;
DROP VIEW dataproduct_with_teamkatalogen_view;

DROP TABLE tk_teams;
DROP TABLE tk_product_areas;

CREATE VIEW dataproduct_view AS(
    SELECT
        dp.id as dp_id,
        dp.name as dp_name,
        dp.description as dp_description,
        dp.group as dp_group,
        dp.created as dp_created,
        dp.last_modified as dp_last_modified,
        dp.slug as dp_slug,
        dp.teamkatalogen_url as teamkatalogen_url,
        dp.team_contact as team_contact,
        dp.team_id as team_id,
        ds.dataproduct_id as ds_dp_id,
        ds.id as ds_id,
        ds.name as ds_name,
        ds.description as ds_description,
        ds.created as ds_created,
        ds.last_modified as ds_last_modified,
        ds.slug as ds_slug,
        ds.keywords as ds_keywords
    FROM
        dataproducts dp
        LEFT JOIN datasets ds ON dp.id = ds.dataproduct_id
);