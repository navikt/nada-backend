package utils

import (
	"fmt"
)

func MakeJoinableViewName(refProjectID, refDatasetID, refTableID string) string {
	return fmt.Sprintf("%v_%v_%v", refProjectID, refDatasetID, refTableID)
}
