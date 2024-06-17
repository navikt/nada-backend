package storage

import (
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/storage/postgres"
)

type Stores struct {
	AccessStorage            service.AccessStorage
	BigQueryStorage          service.BigQueryStorage
	DataProductsStorage      service.DataProductsStorage
	InsightProductStorage    service.InsightProductStorage
	JoinableViewsStorage     service.JoinableViewsStorage
	KeyWordStorage           service.KeywordsStorage
	MetaBaseStorage          service.MetabaseStorage
	PollyStorage             service.PollyStorage
	ProductAreaStorage       service.ProductAreaStorage
	SearchStorage            service.SearchStorage
	StoryStorage             service.StoryStorage
	ThirdPartyMappingStorage service.ThirdPartyMappingStorage
	TokenStorage             service.TokenStorage
}

func NewStores(
	db *database.Repo,
) *Stores {
	return &Stores{
		AccessStorage:            postgres.NewAccessStorage(db),
		BigQueryStorage:          postgres.NewBigQueryStorage(db),
		DataProductsStorage:      postgres.NewDataProductStorage(db),
		InsightProductStorage:    postgres.NewInsightProductStorage(db),
		JoinableViewsStorage:     postgres.NewJoinableViewStorage(db),
		KeyWordStorage:           postgres.NewKeywordsStorage(db),
		MetaBaseStorage:          postgres.NewMetabaseStorage(db),
		PollyStorage:             postgres.NewPollyStorage(db),
		ProductAreaStorage:       postgres.NewProductAreaStorage(db),
		SearchStorage:            postgres.NewSearchStorage(db),
		StoryStorage:             postgres.NewStoryStorage(db),
		ThirdPartyMappingStorage: postgres.NewThirdPartyMappingStorage(db),
		TokenStorage:             postgres.NewTokenStorage(db),
	}
}
