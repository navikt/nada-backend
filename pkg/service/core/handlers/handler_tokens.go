package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/navikt/nada-backend/pkg/errs"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"github.com/rs/zerolog"
)

type TokenHandler struct {
	tokenService      service.TokenService
	tokenForAPIAccess string
	log               zerolog.Logger
}

func (h *TokenHandler) RotateNadaToken(ctx context.Context, r *http.Request, _ any) (*transport.Empty, error) {
	const op errs.Op = "TokenHandler.RotateNadaToken"

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	err := h.tokenService.RotateNadaToken(ctx, user, r.URL.Query().Get("team"))
	if err != nil {
		return nil, errs.E(op, err)
	}

	return &transport.Empty{}, nil
}

func (h *TokenHandler) GetAllTeamTokens(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	authHeaderParts := strings.Split(authHeader, " ")
	if len(authHeaderParts) != 2 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if authHeaderParts[1] != h.tokenForAPIAccess {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	tokenTeamMap, err := h.tokenService.GetNadaTokens(r.Context())
	if err != nil {
		h.log.Error().Err(err).Msg("getting nada tokens")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	payloadBytes, err := json.Marshal(tokenTeamMap)
	if err != nil {
		h.log.Error().Err(err).Msg("marshalling nada token map reponse")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(payloadBytes)
}

func NewTokenHandler(tokenService service.TokenService, tokenForAPIAccess string, log zerolog.Logger) *TokenHandler {
	return &TokenHandler{
		tokenService:      tokenService,
		tokenForAPIAccess: tokenForAPIAccess,
		log:               log,
	}
}
