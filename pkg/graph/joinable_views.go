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
		ID:      jviewDB.ID,
		Name:    jviewDB.Name,
		Created: jviewDB.Created,
		Expires: jviewDB.Expires,
	}
	return jview
}

func (r *queryResolver) JoinableViewWithDatasourceDBToGraph(jviewDB *database.JoinableViewWithDatasource) *models.JoinableViewWithDatasource {

	jview := models.JoinableViewWithDatasource{
		JoinableView: *r.JoinableViewDBToGraph(&jviewDB.JoinableView),
	}
	for _, v := range jviewDB.PseudoDatasources {
		jview.PseudoDatasources = append(jview.PseudoDatasources,
			models.JoinableViewDatasource{
				BigQueryUrl: r.bigquery.MakeBigQueryUrlForJoinableViews(jviewDB.Name, v.ProjectID, v.DatasetID, v.TableID),
				Accessible:  v.Accessible,
				Deleted:     v.Deleted,
			})
	}

	return &jview
}
