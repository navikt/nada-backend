package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/go-chi/chi"
	"github.com/navikt/nada-backend/pkg/tk"
	"github.com/rs/zerolog"
)

var (
	data = flag.String("data", "", "Path to the JSON file to serve")
	port = flag.String("port", "8080", "Port to run the HTTP server on")
)

type Data struct {
	ProductAreas tk.ProductAreas `json:"product_areas"`
	Teams        tk.Teams        `json:"teams"`
}

type Handlers struct {
	data *Data
	log  zerolog.Logger
}

func New(data *Data, log zerolog.Logger) *Handlers {
	return &Handlers{data, log}
}

func (h *Handlers) Log(r *http.Request) {
	body, _ := httputil.DumpRequest(r, true)
	h.log.Info().Msgf("Request received %s %s %s", r.Method, r.URL, string(body))
}

func (h *Handlers) GetProductAreas(w http.ResponseWriter, r *http.Request) {
	h.Log(r)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.data.ProductAreas)
}

func (h *Handlers) GetTeams(w http.ResponseWriter, r *http.Request) {
	h.Log(r)

	productArea := r.URL.Query().Get("productArea")

	teams := h.data.Teams.Content
	if productArea != "" {
		teams = nil
		for _, team := range h.data.Teams.Content {
			if team.ProductAreaID.String() == productArea {
				teams = append(teams, team)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tk.Teams{
		Content: teams,
	})
}

func (h *Handlers) GetTeam(w http.ResponseWriter, r *http.Request) {
	h.Log(r)

	id := chi.URLParam(r, "id")

	for _, team := range h.data.Teams.Content {
		if team.ID.String() == id {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(team)
			return
		}
	}

	http.Error(w, "team not found", http.StatusNotFound)
}

func main() {
	flag.Parse()

	log := zerolog.New(os.Stdout)

	d, err := os.ReadFile(*data)
	if err != nil {
		log.Fatal().Err(err).Msg("opening file")
	}

	data := &Data{}
	err = json.Unmarshal(d, data)
	if err != nil {
		log.Fatal().Err(err).Msg("parsing JSON")
	}

	h := New(data, log)

	r := chi.NewRouter()
	r.Get("/productarea", h.GetProductAreas)
	r.Get("/team", h.GetTeams)
	r.Get("/team/{id}", h.GetTeam)

	log.Printf("Server starting on port %s...", *port)
	err = http.ListenAndServe(":"+*port, r)
	if err != nil {
		log.Fatal().Err(err).Msg("starting server")
	}
}
