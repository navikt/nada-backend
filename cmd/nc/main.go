package main

import (
	"flag"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog"
)

var (
	data = flag.String("data", "", "Path to the JSON file to serve")
	port = flag.Int("port", 8080, "Port to run the HTTP server on")
)

func DefaultHandler(response []byte, log zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := httputil.DumpRequest(r, true)

		log.Info().Msgf("Request received %s %s %s", r.Method, r.URL, string(body))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(response)
	}
}

func main() {
	flag.Parse()

	log := zerolog.New(os.Stdout)

	response, err := os.ReadFile(*data)
	if err != nil {
		log.Fatal().Err(err).Msg("opening file")
	}

	r := chi.NewRouter()
	r.Post("/*", DefaultHandler(response, log))

	log.Printf("Server starting on port %d...", *port)
	err = http.ListenAndServe(":"+strconv.Itoa(*port), r)
	if err != nil {
		log.Fatal().Err(err).Msg("starting server")
	}
}
