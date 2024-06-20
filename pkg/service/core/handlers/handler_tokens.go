package handlers

import (
	"context"
	"encoding/json"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

type tokenHandler struct {
	tokenService   service.TokenService
	teamTokenCreds string
}

func (h *tokenHandler) RotateNadaToken(ctx context.Context, r *http.Request, _ any) (*transport.Empty, error) {
	err := h.tokenService.RotateNadaToken(ctx, r.URL.Query().Get("team"))
	if err != nil {
		return nil, err
	}

	return &transport.Empty{}, nil
}

func (h *tokenHandler) GetAllTeamTokens(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	authHeaderParts := strings.Split(authHeader, " ")
	if len(authHeaderParts) != 2 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if authHeaderParts[1] != h.teamTokenCreds {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	tokenTeamMap, err := h.tokenService.GetNadaTokens(r.Context())
	if err != nil {
		log.WithError(err).Error("getting nada tokens")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	payloadBytes, err := json.Marshal(tokenTeamMap)
	if err != nil {
		log.WithError(err).Error("marshalling nada token map reponse")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(payloadBytes)
}

func NewTokenHandler(tokenService service.TokenService) *tokenHandler {
	return &tokenHandler{
		tokenService: tokenService,
	}
}
