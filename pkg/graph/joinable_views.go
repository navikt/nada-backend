package graph

import (
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *queryResolver) JoinableViewsDBToGraph(jviewsDB []*database.JoinableView) []*models.JoinableView {
	jviews := []*models.JoinableView{}
	for _, v := range jviewsDB {
		jviews = append(jviews, r.JoinableViewDBToGraph(v))
	}
	return jviews
}
func (r *queryResolver) JoinableViewDBToGraph(jviewDB *database.JoinableView) *models.JoinableView {
	jview := &models.JoinableView{
		ID:               jviewDB.ID,
		Name:             jviewDB.Name,
		Created:          jviewDB.Created,
		Expires:          jviewDB.Expires,
		BigQueryViewUrls: []string{},
	}

	for _, v := range jviewDB.PseudoDatasources {
		jview.BigQueryViewUrls = append(jview.BigQueryViewUrls, r.bigquery.MakeBigQueryUrlForJoinableViews(jviewDB.Name, v.ProjectID, v.Dataset, v.Table))
	}
	return jview
}


func (r *queryResolver) JoinableViewWithAccessDBToGraph(jviewDB *database.JoinableViewInDetail) *models.JoinableViewInDetail {
	
	jview := models.JoinableViewInDetail{
		JoinableView: *r.JoinableViewDBToGraph(&jviewDB.JoinableView),
	}
	for _, access:= range jviewDB.AccessToViews{
		jview.AccessToViews = append(jview.AccessToViews, access)
	}
	
	return &jview
}
