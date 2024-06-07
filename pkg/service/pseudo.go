package service

import "context"

func GetPseudoDatasourcesToDelete(ctx context.Context) ([]*BigQuery, error) {
	rows, err := queries.GetPseudoDatasourcesToDelete(ctx)
	if err != nil {
		return nil, err
	}

	pseudoViews := []*BigQuery{}
	for _, d := range rows {
		pseudoViews = append(pseudoViews, &BigQuery{
			ID:            d.ID,
			Dataset:       d.Dataset,
			ProjectID:     d.ProjectID,
			Table:         d.TableName,
			PseudoColumns: d.PseudoColumns,
		})
	}
	return pseudoViews, nil
}
