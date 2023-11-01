-- +goose Up
DROP TABLE dataset_requesters;

-- +goose Down
CREATE TABLE dataset_requesters(
    dataset_id uuid NOT NULL,
    "subject" TEXT NOT NULL,
    PRIMARY KEY (dataset_id, "subject"),
    CONSTRAINT fk_requester_dataset
        FOREIGN KEY (dataset_id)
            REFERENCES datasets (id) ON DELETE CASCADE
);
