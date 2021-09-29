package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/navikt/datakatalogen/backend/api"
	"github.com/navikt/datakatalogen/backend/database"
	"github.com/navikt/datakatalogen/backend/database/gensql"
	"github.com/navikt/datakatalogen/backend/openapi"
)

func main() {
	db, err := sql.Open("postgres", "user=postgres dbname=datakatalogen sslmode=disable password=navikt")
	if err != nil {
		log.Fatal(err)
	}
	repo, _ := database.New(gensql.New(db))
	srv := api.New(repo)
	router := openapi.Handler(srv)
	fmt.Println("Listening on 3000")
	log.Fatal(http.ListenAndServe(":3000", router))
}
