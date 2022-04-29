-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE OR REPLACE FUNCTION update_modified_timestamp() RETURNS TRIGGER AS
$$ BEGIN NEW.last_modified = NOW(); RETURN NEW; END; $$
    LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION f_arr2text(text[])
    RETURNS text LANGUAGE sql IMMUTABLE AS $$SELECT array_to_string($1, ',')$$;

-- +goose Down
DROP FUNCTION f_arr2text(text[]);
DROP FUNCTION update_modified_timestamp();
