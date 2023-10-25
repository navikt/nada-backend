-- +goose Up
CREATE TYPE access_request_status_type AS ENUM ('pending', 'approved', 'denied');

CREATE TABLE polly_documentation
(
    "id"          uuid DEFAULT uuid_generate_v4(),
    "external_id" TEXT NOT NULL,
    "name"        TEXT NOT NULL,
    "url"         TEXT NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE dataproduct_access_request
(
    "id"                     uuid                                 DEFAULT uuid_generate_v4(),
    "dataproduct_id"         uuid                       NOT NULL, 
    "subject"                TEXT                       NOT NULL,
    "owner"                  TEXT                       NOT NULL,
    "polly_documentation_id" uuid,
    "last_modified"          TIMESTAMPTZ                NOT NULL  DEFAULT NOW(),
    "created"                TIMESTAMPTZ                NOT NULL  DEFAULT NOW(),
    "expires"                TIMESTAMPTZ,
    "status"                 access_request_status_type NOT NULL  DEFAULT 'pending',
    "closed"                 TIMESTAMPTZ,
    "granter"                TEXT,
    PRIMARY KEY (id),
    CONSTRAINT fk_requester_dataproduct
        FOREIGN KEY (dataproduct_id)
            REFERENCES dataproducts (id) ON DELETE CASCADE
);

CREATE TRIGGER dataproduct_access_request_set_modified
    BEFORE UPDATE
    ON dataproduct_access_request
    FOR EACH ROW
EXECUTE PROCEDURE update_modified_timestamp();

-- +goose Down
DROP TRIGGER dataproduct_access_request_set_modified ON dataproduct_access_request;
DROP TABLE dataproduct_access_request;
DROP TABLE polly_documentation;
