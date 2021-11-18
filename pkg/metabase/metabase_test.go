package metabase

import (
	"context"
	"fmt"
	"testing"
	"time"
)

var saJSON = `
{
hent meg her:  https://console.cloud.google.com/iam-admin/serviceaccounts/details/108506659724665662385/keys?project=aura-prod-d7e3
}
`

func TestDev(t *testing.T) {
	m := New("https://metabase.dev.intern.nav.no/api", "nada-backend@nav.no", "hint: secret/metabase-sa i nada ns")

	//id, err := m.client.CreateDatabase(context.Background(), "Container resource usage", saJSON, &models.BigQuery{
	//	DataproductID: uuid.New(),
	//	ProjectID:     "aura-prod-d7e3",
	//	Dataset:       "container_resource_usage",
	//	Table:         "data",
	//})
	//if err != nil {
	//	t.Fatal(err)
	//}
	id := "14"
	fmt.Println("New database id", id)

	ctx := context.Background()
	time.Sleep(2 * time.Second) //TODO(jhrv): seems it's unable to hide other tables immediately after creation
	if err := m.HideOtherTables(ctx, id, "data"); err != nil {
		t.Fatal(err)
	}

	tables, err := m.client.Tables(ctx, id)
	if err != nil {
		t.Fatal(err)
	}

	if err := m.client.AutoMapSemanticTypes(ctx, id); err != nil {
		t.Fatal(err)
	}

	for _, t := range tables {
		fmt.Println("Name", t.Name)
		for _, f := range t.Fields {
			fmt.Println("Field.DatabaseType", f.DatabaseType)
			fmt.Println("Field.ID", f.ID)
		}
	}

	//if err := m.client.DeleteDatabase(ctx, id); err != nil {
	//	t.Fatal(err)
	//}

	dbs, err := m.client.Databases(ctx)
	if err != nil {
		t.Error(err)
	}
	for _, db := range dbs {
		fmt.Println(db)
	}
}
