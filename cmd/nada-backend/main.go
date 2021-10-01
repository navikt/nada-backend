package main

import (
	"database/sql"
	"net/http"

	"github.com/navikt/nada-backend/pkg/api"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/openapi"
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
	log.Info("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
