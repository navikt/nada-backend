-- +goose Up

CREATE TABLE dataproduct_requesters(
    dataproduct_id uuid NOT NULL,
    "subject" TEXT NOT NULL,
    PRIMARY KEY (dataproduct_id, "subject"),
    CONSTRAINT fk_requester_dataproduct
        FOREIGN KEY (dataproduct_id)
            REFERENCES dataproducts (id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE dataproduct_requesters;
