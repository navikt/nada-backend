package models

import "github.com/google/uuid"

type MetabaseMetadata struct {
	DataproductID     uuid.UUID
	DatabaseID        int
	PermissionGroupID int
}
