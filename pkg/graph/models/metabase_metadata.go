package models

import (
	"time"

	"github.com/google/uuid"
)

type MetabaseMetadata struct {
	DatasetID         uuid.UUID
	DatabaseID        int
	PermissionGroupID int
	CollectionID      int
	SAEmail           string
	DeletedAt         *time.Time
}
