-- +goose Up
CREATE OR REPLACE FUNCTION update_dataproduct_modified_timestamp() RETURNS TRIGGER AS
$$ BEGIN UPDATE dataproducts SET last_modified = now() WHERE id = NEW.dataproduct_id; RETURN NEW; END; $$
LANGUAGE plpgsql;
