package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/navikt/nada-backend/pkg/api"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/openapi"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

func main() {
	log := logrus.StandardLogger()
	log.SetFormatter(&logrus.JSONFormatter{})

	dbConnDSN := flag.String("db-connection-dsn", fmt.Sprintf("%v?sslmode=disable", getEnv("NAIS_DATABASE_NADA_BACKEND_NADA_URL", "postgres://postgres:postgres@127.0.0.1:5432/nada")), "database connection DSN")
	flag.Parse()

	repo, err := database.New(*dbConnDSN)
	if err != nil {
		log.WithError(err).Fatal("setting up database")
	}

	srv := api.New(repo, log.WithField("subsystem", "api"))
	router := openapi.HandlerWithOptions(srv, openapi.ChiServerOptions{BaseURL: "/api"})
	log.Info("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func getEnv(key, fallback string) string {
	if env := os.Getenv(key); env != "" {
		return env
	}
	return fallback
}
