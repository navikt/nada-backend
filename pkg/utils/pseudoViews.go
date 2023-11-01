package utils

import (
	"fmt"
)

func MakeJoinableViewName(projectID, datasetID, tableID string) string {
	//datasetID will always be same markedsplassen dataset id
	return fmt.Sprintf("%v_%v", projectID, tableID)
}
