package integration

import (
	"context"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/service"
	"testing"
)

var (
	ProductAreaOceanicID   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	ProductAreaOceanicName = "Oceanic"

	ProductAreaCostalID   = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	ProductAreaCostalName = "Costal"

	TeamSeagrassID   = uuid.MustParse("00000000-0000-0000-0000-000000000003")
	TeamSeagrassName = "Seagrass"

	TeamReefID   = uuid.MustParse("00000000-0000-0000-0000-000000000004")
	TeamReefName = "Reef"

	GroupNada = "nada@nav.no"
)

func StorageCreateProductAreasAndTeams(t *testing.T, storage service.ProductAreaStorage) {
	t.Helper()

	pas := []*service.UpsertProductAreaRequest{
		{
			ID:   ProductAreaOceanicID,
			Name: ProductAreaOceanicName,
		},
		{
			ID:   ProductAreaCostalID,
			Name: ProductAreaCostalName,
		},
	}

	teams := []*service.UpsertTeamRequest{
		{
			ID:            TeamSeagrassID,
			ProductAreaID: ProductAreaOceanicID,
			Name:          TeamSeagrassName,
		},
		{
			ID:            TeamReefID,
			ProductAreaID: ProductAreaCostalID,
			Name:          TeamReefName,
		},
	}

	err := storage.UpsertProductAreaAndTeam(context.Background(), pas, teams)
	if err != nil {
		t.Fatalf("creating product areas and teams: %v", err)
	}
}

func NewDataProductBiofuelProduction() service.NewDataproduct {
	return service.NewDataproduct{
		Name:          "Biofuel Production",
		Description:   strToStrPtr("Using seagrass as a feedstock to create renewable biofuels"),
		Group:         GroupNada,
		ProductAreaID: strToStrPtr(ProductAreaOceanicID.String()),
		TeamID:        strToStrPtr(TeamSeagrassID.String()),
	}
}

func NewDataProductAquacultureFeed() service.NewDataproduct {
	return service.NewDataproduct{
		Name:          "Aquaculture Feed",
		Description:   strToStrPtr("Producing high-nutrient feed for aquaculture industries from processed seagrass"),
		Group:         GroupNada,
		ProductAreaID: strToStrPtr(ProductAreaOceanicID.String()),
		TeamID:        strToStrPtr(TeamSeagrassID.String()),
	}
}

func NewDataProductReefMonitoring() service.NewDataproduct {
	return service.NewDataproduct{
		Name:          "Reef Monitoring Equipment",
		Description:   strToStrPtr("Advanced sensors and monitoring devices for continuous assessment"),
		Group:         GroupNada,
		ProductAreaID: strToStrPtr(ProductAreaCostalID.String()),
		TeamID:        strToStrPtr(TeamReefID.String()),
	}
}

func NewDataProductProtectiveBarriers() service.NewDataproduct {
	return service.NewDataproduct{
		Name:          "Protective Barriers",
		Description:   strToStrPtr("Physical barriers to protect coral reefs from human activity"),
		Group:         GroupNada,
		ProductAreaID: strToStrPtr(ProductAreaCostalID.String()),
		TeamID:        strToStrPtr(TeamReefID.String()),
	}
}

func StorageCreateDataproduct(t *testing.T, storage service.DataProductsStorage, ndp service.NewDataproduct) *service.DataproductMinimal {
	t.Helper()

	dp, err := storage.CreateDataproduct(context.Background(), ndp)
	if err != nil {
		t.Fatalf("creating dataproduct: %v", err)
	}

	return dp
}
