package integration

import (
	"context"
	"testing"

	"github.com/goccy/bigquery-emulator/types"
	bigQueryEmulator "github.com/navikt/nada-backend/pkg/bq/emulator"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/service"
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

	NameNada      = "nada"
	GroupNada     = "nada@nav.no"
	NaisTeamNada  = "nada"
	NameAllUsers  = "all-users"
	GroupAllUsers = "all-users@nav.no"

	Project       = "test-project"
	Location      = "europe-north1"
	PseudoDataSet = "pseudo-test-dataset"

	TestUser = &service.User{
		Name:  NameNada,
		Email: GroupNada,
		GoogleGroups: []service.Group{
			{
				Name:  NameNada,
				Email: GroupNada,
			},
			{
				Name:  NameAllUsers,
				Email: GroupAllUsers,
			},
		},
	}
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
		ProductAreaID: &ProductAreaOceanicID,
		TeamID:        &TeamSeagrassID,
	}
}

func NewDatasetBiofuelConsumptionRatesSchema() []*bigQueryEmulator.Dataset {
	return []*bigQueryEmulator.Dataset{
		{
			DatasetID: "biofuel",
			TableID:   "consumption_rates",
			Columns: []*types.Column{
				bigQueryEmulator.ColumnRequired("id"),
				bigQueryEmulator.ColumnNullable("fuel_type"),
				bigQueryEmulator.ColumnNullable("consumption_rate"),
				bigQueryEmulator.ColumnNullable("unit"),
			},
		},
		{
			DatasetID: PseudoDataSet,
		},
	}
}

func NewDatasetBiofuelConsumptionRates(dataProductID uuid.UUID) service.NewDataset {
	dataset := NewDatasetBiofuelConsumptionRatesSchema()[0]

	return service.NewDataset{
		DataproductID: dataProductID,
		Name:          "Biofuel Consumption Rates",
		Description:   strToStrPtr("Consumption rates of biofuels in the transportation sector"),
		Keywords:      []string{"biofuel", "consumption", "rates"},
		BigQuery: service.NewBigQuery{
			ProjectID: Project,
			Dataset:   dataset.DatasetID,
			Table:     dataset.TableID,
		},
		Pii: service.PiiLevelNone,
	}
}

func NewDataProductAquacultureFeed() service.NewDataproduct {
	return service.NewDataproduct{
		Name:          "Aquaculture Feed",
		Description:   strToStrPtr("Producing high-nutrient feed for aquaculture industries from processed seagrass"),
		Group:         GroupNada,
		ProductAreaID: &ProductAreaOceanicID,
		TeamID:        &TeamSeagrassID,
	}
}

func NewDataProductReefMonitoring() service.NewDataproduct {
	return service.NewDataproduct{
		Name:          "Reef Monitoring Equipment",
		Description:   strToStrPtr("Advanced sensors and monitoring devices for continuous assessment"),
		Group:         GroupNada,
		ProductAreaID: &ProductAreaCostalID,
		TeamID:        &TeamReefID,
	}
}

func NewDataProductProtectiveBarriers() service.NewDataproduct {
	return service.NewDataproduct{
		Name:          "Protective Barriers",
		Description:   strToStrPtr("Physical barriers to protect coral reefs from human activity"),
		Group:         GroupNada,
		ProductAreaID: &ProductAreaCostalID,
		TeamID:        &TeamReefID,
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
