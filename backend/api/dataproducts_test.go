package api_test

import (
	"github.com/navikt/datakatalogen/backend/api"
	"testing"

	"github.com/navikt/datakatalogen/backend/iam"
	"github.com/stretchr/testify/assert"
)

func TestValidation(t *testing.T) {
	missingType := map[string]string{"no": "type"}
	err := api.ValidateDatastore(missingType)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "no type defined")

	invalidType := map[string]string{"type": "nonexistent"}
	err = api.ValidateDatastore(invalidType)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "unknown datastore type: nonexistent")

	validBucket := map[string]string{"type": iam.BucketType, "project_id": "ap", "bucket_id": "x"}
	assert.NoError(t, api.ValidateDatastore(validBucket))
	invalidBucket := map[string]string{"type": iam.BucketType, "project_id": "ap"}
	err = api.ValidateDatastore(invalidBucket)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "missing key: bucket_id")

	validBigQuery := map[string]string{"type": iam.BigQueryType, "project_id": "pi", "dataset_id": "did", "resource_id": "rid"}
	assert.NoError(t, api.ValidateDatastore(validBigQuery))

	invalidBigQuery := map[string]string{"type": iam.BigQueryType, "project_id": "pi", "dataset_id": "did"}
	err = api.ValidateDatastore(invalidBigQuery)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "missing key: resource_id")
}
