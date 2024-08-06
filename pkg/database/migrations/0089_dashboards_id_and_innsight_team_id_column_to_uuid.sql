-- +goose Up
ALTER TABLE dashboards
    ALTER COLUMN id
        SET DATA TYPE UUID
        USING CASE
                  WHEN id ~* '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$' THEN id::UUID
        END;

DROP VIEW insight_product_with_teamkatalogen_view;
ALTER TABLE insight_product
    ALTER COLUMN team_id
        SET DATA TYPE UUID
        USING CASE
            WHEN team_id ~* '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$' THEN team_id::UUID
        END;
CREATE VIEW insight_product_with_teamkatalogen_view AS(
SELECT isp.*, tkt.name as team_name, tkpa.name as pa_name FROM insight_product isp LEFT JOIN 
	(tk_teams tkt LEFT JOIN tk_product_areas tkpa
	ON tkt.product_area_id = tkpa.id)
	ON isp.team_id = tkt.id
);

-- +goose Down
ALTER TABLE dashboards
    ALTER COLUMN id
        SET DATA TYPE TEXT
        USING id::TEXT;

DROP VIEW insight_product_with_teamkatalogen_view;
ALTER TABLE insight_product
    ALTER COLUMN team_id
        SET DATA TYPE TEXT
        USING team_id::TEXT;
CREATE VIEW insight_product_with_teamkatalogen_view AS(
SELECT isp.*, tkt.name as team_name, tkpa.name as pa_name FROM insight_product isp LEFT JOIN 
	(tk_teams tkt LEFT JOIN tk_product_areas tkpa
	ON tkt.product_area_id = tkpa.id)
	ON isp.team_id = tkt.id::TEXT
);
