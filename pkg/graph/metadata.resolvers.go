package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
)

func (r *mutationResolver) TriggerMetadataSync(ctx context.Context) (bool, error) {
	bqs, err := r.repo.GetBigqueryDatasources(ctx)
	if err != nil {
		return false, err
	}

	var errs errorList

	for _, bq := range bqs {
		err := r.UpdateMetadata(ctx, bq)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) != 0 {
		return false, errs
	}
	return true, nil
}
