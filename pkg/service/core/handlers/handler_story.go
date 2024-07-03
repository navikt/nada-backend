package handlers

import (
	"context"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/amplitude"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	"github.com/navikt/nada-backend/pkg/service/core/transport"
	"github.com/rs/zerolog"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

const (
	ContextKeyTeam      = "team"
	ContextKeyTeamEmail = "team_email"
	ContextKeyNadaToken = "nada_token"
)

type StoryHandler struct {
	storyService    service.StoryService
	tokenService    service.TokenService
	amplitudeClient amplitude.Amplitude
	log             zerolog.Logger
}

func (h *StoryHandler) DeleteStory(ctx context.Context, _ *http.Request, _ any) (*service.Story, error) {
	const op errs.Op = "StoryHandler.DeleteStory"

	id, err := uuid.Parse(chi.URLParamFromCtx(ctx, "id"))
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	user := auth.GetUser(ctx)

	story, err := h.storyService.DeleteStory(ctx, user, id)
	if err != nil {
		return nil, err
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

	story, err := h.storyService.UpdateStory(ctx, user, id, in)
	if err != nil {
		return nil, err
	}

	return story, nil
}

func (h *StoryHandler) CreateStory(ctx context.Context, r *http.Request, newStory *service.NewStory) (*service.Story, error) {
	const op errs.Op = "StoryHandler.CreateStory"

	err := newStory.Validate()
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, err)
	}

	files, err := filesFromRequest(r)
	if err != nil {
		return nil, err
	}

	user := auth.GetUser(ctx)

	story, err := h.storyService.CreateStory(ctx, user.Email, newStory, files)
	if err != nil {
		return nil, err
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
		return nil, err
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

	files, err := filesFromRequest(r)
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, errs.Parameter("files"), err)
	}

	raw := r.Context().Value(ContextKeyTeamEmail)
	teamEmail, ok := raw.(string)
	if !ok {
		return nil, errs.E(errs.Internal, op, fmt.Errorf("team not found in context"))
	}

	err = h.storyService.RecreateStoryFiles(ctx, id, teamEmail, files)
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

	files, err := filesFromRequest(r)
	if err != nil {
		return nil, errs.E(errs.InvalidRequest, op, errs.Parameter("files"), err)
	}

	raw := r.Context().Value(ContextKeyTeamEmail)
	teamEmail, ok := raw.(string)
	if !ok {
		return nil, errs.E(errs.Internal, op, fmt.Errorf("team not found in context"))
	}

	err = h.storyService.AppendStoryFiles(ctx, id, teamEmail, files)
	if err != nil {
		return nil, errs.E(op, err)
	}

	return &transport.Empty{}, nil
}

func (h *StoryHandler) NadaTokenMiddleware(next http.Handler) http.Handler {
	const op errs.Op = "StoryHandler.NadaTokenMiddleware"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		splitToken := strings.Split(token, "Bearer ")
		token = splitToken[1]
		if len(token) == 0 {
			errs.HTTPErrorResponse(w, h.log, errs.E(errs.Unauthenticated, op, errs.Parameter("nada_token"), fmt.Errorf("no token provided")))
		}

		valid, err := h.tokenService.ValidateToken(r.Context(), token)
		if err != nil {
			errs.HTTPErrorResponse(w, h.log, errs.E(errs.Internal, op, err))
		}

		if !valid {
			errs.HTTPErrorResponse(w, h.log, errs.E(errs.Unauthenticated, op, errs.Parameter("nada_token"), fmt.Errorf("invalid nada token")))
		}

		team, err := h.tokenService.GetTeamFromNadaToken(r.Context(), token)
		if err != nil {
			errs.HTTPErrorResponse(w, h.log, errs.E(errs.Unauthorized, op, err))
		}

		ctx := context.WithValue(r.Context(), ContextKeyTeam, team)
		ctx = context.WithValue(ctx, ContextKeyTeamEmail, team+"@nav.no")
		ctx = context.WithValue(ctx, ContextKeyNadaToken, token)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// FIXME: move into a separate file, make it testable
func filesFromRequest(r *http.Request) ([]*service.UploadFile, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, fmt.Errorf("creating multipart reader: %w", err)
	}

	var files []*service.UploadFile
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading next part: %w", err)
		}

		if part.FileName() == "" {
			continue
		}

		data, err := io.ReadAll(part)
		if err != nil {
			return nil, fmt.Errorf("reading part data: %w", err)
		}

		fileFullPath := part.FileName()

		// try to extract full path from content-disposition header
		_, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		if err == nil {
			pathInCDHeader := params["name"]
			if pathInCDHeader != "" {
				fileFullPath = pathInCDHeader
			}
		}

		files = append(files, &service.UploadFile{
			Path: fileFullPath,
			Data: data,
		})
	}

	return files, nil
}

func NewStoryHandler(storyService service.StoryService, tokenService service.TokenService, log zerolog.Logger) *StoryHandler {
	return &StoryHandler{
		storyService: storyService,
		tokenService: tokenService,
		log:          log,
	}
}
