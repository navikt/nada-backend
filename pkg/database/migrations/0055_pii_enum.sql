-- +goose Up
CREATE TYPE pii_level AS ENUM ('sensitive', 'anonymised', 'none');

ALTER TABLE datasets 
    ALTER COLUMN pii TYPE pii_level 
    USING CASE 
        WHEN pii = FALSE 
            THEN 'none'::pii_level
            ELSE 'sensitive'::pii_level
    END;

-- +goose Down
ALTER TABLE datasets 
    ALTER COLUMN pii TYPE BOOLEAN 
    USING CASE 
        WHEN pii = 'none' OR pii = 'anonymised' 
            THEN FALSE 
            ELSE TRUE 
    END;