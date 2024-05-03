-- +goose Up
ALTER TABLE nada_tokens DROP CONSTRAINT nada_tokens_token_key;

WITH without_naisteam_prefix AS (
  SELECT REPLACE(team, 'nais-team-', '') as team, token FROM nada_tokens 
  WHERE team LIKE 'nais-team-%'
)

INSERT INTO nada_tokens (team,token) (
  SELECT team, token from without_naisteam_prefix
) ON CONFLICT (team) DO
UPDATE
SET token=EXCLUDED.token;

DELETE FROM nada_tokens WHERE team LIKE 'nais-team-%';

ALTER TABLE nada_tokens ADD CONSTRAINT nada_tokens_token_key UNIQUE (token);
