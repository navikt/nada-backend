package main

import (
	"database/sql"
	"net/http"

	"github.com/navikt/datakatalogen/backend/api"
	"github.com/navikt/datakatalogen/backend/database"
	"github.com/navikt/datakatalogen/backend/database/gensql"
	"github.com/navikt/datakatalogen/backend/openapi"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.StandardLogger()
	log.SetFormatter(&logrus.JSONFormatter{})

	db, err := sql.Open("postgres", "user=postgres dbname=datakatalogen sslmode=disable password=navikt")
	if err != nil {
		log.Fatal(err)
	}
	repo, _ := database.New(gensql.New(db))
	srv := api.New(repo, log.WithField("subsystem", "api"))
	router := openapi.HandlerWithOptions(srv, openapi.ChiServerOptions{BaseURL: "/api"})
	log.Info("Listening on :3000")
	log.Fatal(http.ListenAndServe("127.0.0.1:3000", router))
}
