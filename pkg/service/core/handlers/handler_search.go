package handlers

import (
	"context"
	"github.com/navikt/nada-backend/pkg/service"
	"net/http"
	"strconv"
	"strings"
)

type SearchHandler struct {
	service service.SearchService
}

func (h *SearchHandler) Search(ctx context.Context, r *http.Request, _ any) (*service.SearchResult, error) {
	searchOptions, err := parseSearchOptionsFromRequest(r)
	if err != nil {
		return nil, nil
	}

	return h.service.Search(ctx, searchOptions)
}

func parseSearchOptionsFromRequest(r *http.Request) (*service.SearchOptions, error) {
	query := r.URL.Query()

	options := service.SearchOptions{}

	// Parse 'text' parameter
	if text, ok := query["text"]; ok && len(text) > 0 {
		options.Text = text[0]
	}

	// Parse 'keywords' parameter
	if keywords, ok := query["keywords"]; ok && len(keywords) > 0 {
		options.Keywords = strings.Split(keywords[0], ",")
	}

	// Parse 'groups' parameter
	if groups, ok := query["groups"]; ok && len(groups) > 0 {
		options.Groups = strings.Split(groups[0], ",")
	}

	// Parse 'teamIDs' parameter
	if teamIDs, ok := query["teamIDs"]; ok && len(teamIDs) > 0 {
		options.TeamIDs = strings.Split(teamIDs[0], ",")
	}

	// Parse 'services' parameter
	if services, ok := query["services"]; ok && len(services) > 0 {
		options.Services = strings.Split(services[0], ",")
	}

	// Parse 'types' parameter
	if types, ok := query["types"]; ok && len(types) > 0 {
		options.Types = strings.Split(types[0], ",")
	}

	// Parse 'limit' parameter
	if limit, ok := query["limit"]; ok && len(limit) > 0 {
		limitVal, err := strconv.Atoi(limit[0])
		if err != nil {
			return nil, err // Handle or return an error appropriately
		}
		options.Limit = &limitVal
	}

	// Parse 'offset' parameter
	if offset, ok := query["offset"]; ok && len(offset) > 0 {
		offsetVal, err := strconv.Atoi(offset[0])
		if err != nil {
			return nil, err // Handle or return an error appropriately
		}
		options.Offset = &offsetVal
	}

	return &options, nil
}

func NewSearchHandler(service service.SearchService) *SearchHandler {
	return &SearchHandler{service: service}
}
