-- +goose Up

CREATE TABLE dataproduct_access (
	id uuid DEFAULT uuid_generate_v4(),
	dataproduct_id uuid NOT NULL,
	"subject"      TEXT NOT NULL,
	granter 	   TEXT NOT NULL,
	expires        TIMESTAMPTZ,
	created        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	revoked        TIMESTAMPTZ,
	PRIMARY KEY (id),
	CONSTRAINT fk_access_dataproduct
			FOREIGN KEY (dataproduct_id)
					REFERENCES dataproducts (id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE dataproduct_access;
