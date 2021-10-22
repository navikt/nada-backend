.PHONY: test integration-test local-with-auth local linux-build docker-build docker-push run-postgres-test stop-postgres-test install-sqlc install-oapi-codegen
DATE = $(shell date "+%Y-%m-%d")
LAST_COMMIT = $(shell git --no-pager log -1 --pretty=%h)
VERSION ?= $(DATE)-$(LAST_COMMIT)
LDFLAGS := -X github.com/navikt/nada-backend/backend/version.Revision=$(shell git rev-parse --short HEAD) -X github.com/navikt/nada-backend/backend/version.Version=$(VERSION)
APP = nada-backend
SQLC_VERSION ?= "v1.10.0"
OAPI_CODEGEN_VERSION ?= "v1.8.2"
# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
	GOBIN=$(shell go env GOPATH)/bin
else
	GOBIN=$(shell go env GOBIN)
endif

test:
	go test ./... -count=1

integration-test:
	go test ./... -count=1 -tags=integration_test

local-with-auth:
	go run ./cmd/nada-backend \
	--oauth2-client-id=$(shell kubectl get --context=dev-gcp --namespace=nada secret/google-oauth -o jsonpath='{.data.CLIENT_ID}' | base64 -d) \
	--oauth2-client-secret=$(shell kubectl get --context=dev-gcp --namespace=nada secret/google-oauth -o jsonpath='{.data.CLIENT_SECRET}' | base64 -d) \
	--teams-token=$(shell kubectl get secret --context=dev-gcp --namespace=nada github-read-token -o jsonpath='{.data.GITHUB_READ_TOKEN}' | base64 -d) \
	--bind-address=127.0.0.1:8080 \
	--hostname=localhost \
	--service-account-file=./test-sa.json \
	--google-admin-subject=johnny.horvi@nav.no \
	--log-level=debug

local:
	go run ./cmd/nada-backend \
	--teams-token=$(shell kubectl get secret --context=dev-gcp --namespace=nada github-read-token -o jsonpath='{.data.GITHUB_READ_TOKEN}' | base64 -d) \
	--bind-address=127.0.0.1:8080 \
	--hostname=localhost \
	--mock-auth \
	--skip-metadata-sync \
	--log-level=debug

migrate:
	go run github.com/pressly/goose/v3/cmd/goose -dir ./pkg/database/migrations postgres "user=postgres dbname=nada sslmode=disable password=postgres" up

generate-graphql:
	go get -d github.com/99designs/gqlgen@latest && go run github.com/99designs/gqlgen generate

generate-sql:
	cd pkg && $(GOBIN)/sqlc generate

generate: generate-sql generate-graphql

linux-build:
	go build -a -installsuffix cgo -o $(APP) -ldflags "-s $(LDFLAGS)" ./cmd/nada-backend

docker-build:
	docker image build -t ghcr.io/navikt/$(APP):$(VERSION) -t ghcr.io/navikt/$(APP):latest .

docker-push:
	docker image push ghcr.io/navikt/$(APP):$(VERSION)
	docker image push ghcr.io/navikt/$(APP):latest

install-oapi-codegen:
	go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION)

install-sqlc:
	go install github.com/kyleconroy/sqlc/cmd/sqlc@$(SQLC_VERSION)
