-- +goose Up
DROP TABLE sessions;

-- +goose Down
CREATE TABLE sessions (
  "token" TEXT NOT NULL,
	"email" TEXT NOT NULL,
	"name" TEXT NOT NULL,
	"created" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	"expires" TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (token)
);