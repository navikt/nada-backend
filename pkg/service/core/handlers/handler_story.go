package handlers

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/parser"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"github.com/rs/zerolog"
)

type ContextKeyType string

const (
	ContextKeyTeam      ContextKeyType = "team"
	ContextKeyTeamEmail ContextKeyType = "team_email"
	ContextKeyNadaToken ContextKeyType = "nada_token"
	FormNameNewStory                   = "nada-backend-new-story"
)

type StoryHandler struct {
	storyService service.StoryService
	tokenService service.TokenService
	log          zerolog.Logger
	emailSuffix  string
}

func (h *StoryHandler) DeleteStory(ctx context.Context, _ *http.Request, _ any) (*service.Story, error) {
	const op errs.Op = "StoryHandler.DeleteStory"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	story, err := h.storyService.DeleteStory(ctx, user, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return story, nil
}

func (h *StoryHandler) UpdateStory(ctx context.Context, _ *http.Request, in service.UpdateStoryDto) (*service.Story, error) {
	const op errs.Op = "StoryHandler.UpdateStory"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	story, err := h.storyService.UpdateStory(ctx, user, id, in)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return story, nil
}

func (h *StoryHandler) CreateStory(ctx context.Context, r *http.Request, _ any) (*service.Story, error) {
	const op errs.Op = "StoryHandler.CreateStory"

	user := auth.GetUser(ctx)
	if user == nil {
		return nil, errs.E(errs.Unauthenticated, op, errs.Str("no user in context"))
	}

	p, err := parser.MultipartFormFromRequest(r)
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	err = p.Process([]string{FormNameNewStory})
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	newStory := &service.NewStory{}

	err = p.DeserializedObject(FormNameNewStory, newStory)
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	err = newStory.Validate()
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	files := p.Files()

	uploadFiles := make([]*service.UploadFile, len(files))
	for i, file := range files {
		uploadFiles[i] = &service.UploadFile{
			Path:       file.Path,
			ReadCloser: file.Reader,
		}
	}

	story, err := h.storyService.CreateStory(ctx, user.Email, newStory, uploadFiles)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return story, nil
}

func (h *StoryHandler) GetStory(ctx context.Context, _ *http.Request, _ any) (*service.Story, error) {
	const op errs.Op = "StoryHandler.GetStory"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("parsing id: %w", err))
	}

	story, err := h.storyService.GetStory(ctx, id)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return story, nil
}

func (h *StoryHandler) GetIndex(ctx context.Context, r *http.Request, _ any) (*transport.Redirect, error) {
	const op errs.Op = "StoryHandler.GetIndex"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, fmt.Errorf("parsing id: %w", err))
	}

	index, err := h.storyService.GetIndexHtmlPath(ctx, id.String())
	if err != nil {
		return nil, errs.E(op, err)
	}

	return transport.NewRedirect(index, r), nil
}

func (h *StoryHandler) GetObject(ctx context.Context, r *http.Request, _ any) (*transport.ByteWriter, error) {
	const op errs.Op = "StoryHandler.GetObject"

	pathParts := strings.Split(r.URL.Path, "/")
	objPath := strings.Join(pathParts[2:], "/")

	obj, err := h.storyService.GetObject(ctx, objPath)
	if err != nil {
		return nil, errs.E(op, err)
	}

	contentType := ""
	switch filepath.Ext(obj.Name) {
	case ".html":
		contentType = "text/html"
	case ".js":
		contentType = "text/javascript"
	case ".css":
		contentType = "text/css"
	default:
		contentType = obj.Attrs.ContentType
	}

	return transport.NewByteWriter(contentType, obj.Attrs.ContentEncoding, obj.Data), nil
}

func (h *StoryHandler) CreateStoryForTeam(ctx context.Context, r *http.Request, newStory *service.NewStory) (*service.Story, error) {
	const op errs.Op = "StoryHandler.CreateStoryForTeam"

	raw := r.Context().Value(ContextKeyTeamEmail)
	teamEmail, ok := raw.(string)
	if !ok {
		return nil, errs.E(errs.Internal, op, fmt.Errorf("team not found in context"))
	}

	newStory.Group = teamEmail
	if newStory.Keywords == nil {
		newStory.Keywords = []string{}
	}

	err := newStory.Validate()
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	story, err := h.storyService.CreateStoryWithTeamAndProductArea(ctx, teamEmail, newStory)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return story, nil
}

func (h *StoryHandler) RecreateStoryFiles(ctx context.Context, r *http.Request, _ any) (*transport.Empty, error) {
	const op errs.Op = "StoryHandler.RecreateStoryFiles"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, errs.Parameter("id"), fmt.Errorf("parsing id: %w", err))
	}

	p, err := parser.MultipartFormFromRequest(r)
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	err = p.Process(nil)
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	files := p.Files()

	uploadedFiles := make([]*service.UploadFile, len(files))
	for i, file := range files {
		uploadedFiles[i] = &service.UploadFile{
			Path:       file.Path,
			ReadCloser: file.Reader,
		}
	}

	raw := r.Context().Value(ContextKeyTeamEmail)
	teamEmail, ok := raw.(string)
	if !ok {
		return nil, errs.E(errs.Internal, op, fmt.Errorf("team not found in context"))
	}

	err = h.storyService.RecreateStoryFiles(ctx, id, teamEmail, uploadedFiles)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return &transport.Empty{}, nil
}

func (h *StoryHandler) AppendStoryFiles(ctx context.Context, r *http.Request, _ any) (*transport.Empty, error) {
	const op errs.Op = "StoryHandler.AppendStoryFiles"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, errs.Parameter("id"), fmt.Errorf("parsing id: %w", err))
	}

	p, err := parser.MultipartFormFromRequest(r)
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	err = p.Process(nil)
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	files := p.Files()

	uploadedFiles := make([]*service.UploadFile, len(files))
	for i, file := range files {
		uploadedFiles[i] = &service.UploadFile{
			Path:       file.Path,
			ReadCloser: file.Reader,
		}
	}

	raw := r.Context().Value(ContextKeyTeamEmail)
	teamEmail, ok := raw.(string)
	if !ok {
		return nil, errs.E(errs.Internal, op, fmt.Errorf("team not found in context"))
	}

	err = h.storyService.AppendStoryFiles(ctx, id, teamEmail, uploadedFiles)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return &transport.Empty{}, nil
}

func (h *StoryHandler) NadaTokenMiddleware(next http.Handler) http.Handler {
	const op errs.Op = "StoryHandler.NadaTokenMiddleware"

	type Data struct {
		team  string
		token string
		email string
	}

	fn := func(r *http.Request) (*Data, error) {
		token, err := parser.BearerTokenFromRequest(parser.HeaderAuthorization, r)
		if err != nil {
			return nil, errs.E(errs.Unauthenticated, op, errs.Parameter("nada_token"), err)
		}

		valid, err := h.tokenService.ValidateToken(r.Context(), token)
		if err != nil {
			return nil, errs.E(errs.Internal, op, err)
		}

		if !valid {
			return nil, errs.E(errs.Unauthorized, op, errs.Parameter("nada_token"), fmt.Errorf("token not valid"))
		}

		team, err := h.tokenService.GetTeamFromNadaToken(r.Context(), token)
		if err != nil {
			return nil, errs.E(errs.Unauthorized, op, err)
		}

		return &Data{
			team:  team,
			token: token,
			email: fmt.Sprintf("%s%s", team, h.emailSuffix),
		}, nil
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d, err := fn(r)
		if err != nil {
			errs.HTTPErrorResponse(w, h.log, err)
			return
		}

		ctx := context.WithValue(r.Context(), ContextKeyTeam, d.team)
		ctx = context.WithValue(ctx, ContextKeyTeamEmail, d.email)
		ctx = context.WithValue(ctx, ContextKeyNadaToken, d.token)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func NewStoryHandler(emailSuffix string, storyService service.StoryService, tokenService service.TokenService, log zerolog.Logger) *StoryHandler {
	return &StoryHandler{
		storyService: storyService,
		tokenService: tokenService,
		log:          log,
		emailSuffix:  emailSuffix,
	}
}
