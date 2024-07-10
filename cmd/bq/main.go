package main

import (
	"context"
	"flag"
	"github.com/goccy/bigquery-emulator/server"
	"github.com/navikt/nada-backend/pkg/bq/emulator"
	"github.com/rs/zerolog"
	"os"
	"os/signal"
	"syscall"
)

var projectID = flag.String("project", "test", "project id")
var dataYAML = flag.String("data", "", "data yaml file")
var port = flag.String("port", "8080", "port")

func main() {
	flag.Parse()

	log := zerolog.New(os.Stdout)

	log.Info().Msg("Starting big query emulator")

	e := emulator.New(log)
	defer e.Cleanup()

	e.WithSource(*projectID, server.YAMLSource(*dataYAML))

	mocks, err := emulator.PolicyMocksFromDataYAML(*dataYAML, log)
	if err != nil {
		log.Fatal().Err(err).Msg("creating policy mocks")
	}

	e.EnableMock(true, log, mocks...)

	log.Info().Msgf("Big query emulator started on %s", *port)

	if err := e.Serve(context.Background(), *port, "8081"); err != nil {
		log.Fatal().Err(err).Msg("serving big query emulator")
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Info().Msg("Received Ctrl-C, shutting down big query emulator")
}
