DATE = $(shell date "+%Y-%m-%d")
LAST_COMMIT = $(shell git --no-pager log -1 --pretty=%h)
VERSION ?= $(DATE)-$(LAST_COMMIT)
LDFLAGS := -X github.com/navikt/nada-backend/backend/version.Revision=$(shell git rev-parse --short HEAD) -X github.com/navikt/nada-backend/backend/version.Version=$(VERSION)
APP = nada-backend
SQLC_VERSION ?= "v1.23.0"

# Directories
#
# All of the following directories can be
# overwritten. If this is done, it is
# only recommended to change the BUILD_DIR
# option.
BUILD_DIR     := build-output
RELEASE_DIR   := $(BUILD_DIR)/release

$(BUILD_DIR):
	-mkdir $(BUILD_DIR)

$(RELEASE_DIR): | $(BUILD_DIR)
	-mkdir $(RELEASE_DIR)

GOPATH  := $(shell go env GOPATH)
GOCACHE := $(shell go env GOCACHE)
GOBIN   ?= $(GOPATH)/bin

GO := $(shell command -v go 2> /dev/null)
ifndef GO
$(error go is required, please install)
endif

DOCKER_COMPOSE := $(shell if command -v docker-compose > /dev/null 2>&1; then echo "docker-compose"; elif command -v docker > /dev/null 2>&1 && docker compose version > /dev/null 2>&1; then echo "docker compose"; else echo ""; fi)

ifndef DOCKER_COMPOSE
$(error "Neither docker-compose nor docker compose command is available, please install Docker")
endif

-include .env

test:
	CGO_ENABLED=1 CXX=clang++ CC=clang CXXFLAGS=-Wno-everything go test ./... -count=1
.PHONY: test

build: $(RELEASE_DIR)
	@echo "Building cmd applications..."
	@CGO_ENABLED=1 CXX=clang++ CC=clang go mod tidy
	@for d in cmd/*; do \
		app=$$(basename $$d); \
		echo "Building $$app..."; \
		CGO_ENABLED=1 CXX=clang++ CC=clang CGO_CXXFLAGS=-Wno-everything CGO_LDFLAGS=-Wno-everything $(GO) build -o $(RELEASE_DIR)/$$app ./$$d; \
	done
	@echo "Build complete. Binaries are located in $(RELEASE_DIR)"
.PHONY: build

env:
	@echo "Re-creating .env file..."
	@echo "NADA_OAUTH_CLIENT_ID=$(shell kubectl get --context=dev-gcp --namespace=nada `kubectl get secret --context=dev-gcp --namespace=nada --sort-by='{.metadata.creationTimestamp}' -l app=nada-backend,type=azurerator.nais.io -o name | tail -1` -o jsonpath='{.data.AZURE_APP_CLIENT_ID}' | base64 -d)" > .env
	@echo "NADA_OAUTH_CLIENT_SECRET=$(shell kubectl get --context=dev-gcp --namespace=nada `kubectl get secret --context=dev-gcp --namespace=nada --sort-by='{.metadata.creationTimestamp}' -l app=nada-backend,type=azurerator.nais.io -o name | tail -1` -o jsonpath='{.data.AZURE_APP_CLIENT_SECRET}' | base64 -d)" >> .env
	@echo "NADA_OAUTH_TENANT_ID=$(shell kubectl get --context=dev-gcp --namespace=nada `kubectl get secret --context=dev-gcp --namespace=nada --sort-by='{.metadata.creationTimestamp}' -l app=nada-backend,type=azurerator.nais.io -o name | tail -1` -o jsonpath='{.data.AZURE_APP_TENANT_ID}' | base64 -d)" >> .env
	@echo "NADA_NAIS_CONSOLE_API_KEY=\"$(shell kubectl get secret --context=dev-gcp --namespace=nada nada-backend-secret -o jsonpath='{.data.NADA_NAIS_CONSOLE_API_KEY}' | base64 -d)\"" >> .env
	@echo "NADA_AMPLITUDE_API_KEY=$(shell kubectl get secret --context=dev-gcp --namespace=nada nada-backend-secret -o jsonpath='{.data.NADA_AMPLITUDE_API_KEY}' | base64 -d)" >> .env
	@echo "NADA_SLACK_WEBHOOK_URL=$(shell kubectl get secret --context=dev-gcp --namespace=nada nada-backend-secret -o jsonpath='{.data.NADA_SLACK_WEBHOOK_URL}' | base64 -d)" >> .env
	@echo "NADA_SLACK_TOKEN=$(shell kubectl get secret --context=dev-gcp --namespace=nada nada-backend-secret -o jsonpath='{.data.NADA_SLACK_TOKEN}' | base64 -d)" >> .env

    # Fetch metabase enterprise edition embedding token, so we get metabase ee locally
    # - https://www.metabase.com/docs/v0.49/configuring-metabase/environment-variables#mb_premium_embedding_token
	@echo "MB_PREMIUM_EMBEDDING_TOKEN=$(shell kubectl get secret --context=dev-gcp --namespace=nada metabase -o jsonpath='{.data.MB_PREMIUM_EMBEDDING_TOKEN}' | base64 -d)" >> .env
.PHONY: env

test-sa:
	@echo "Fetching service account credentials..."
	$(shell kubectl get --context=dev-gcp --namespace=nada secret/nada-backend-google-credentials -o json | jq -r '.data."sa.json"' | base64 -d > test-sa.json)
.PHONY: test-sa

metabase-sa:
	@echo "Fetching metabase service account credentials..."
	$(shell kubectl get --context=dev-gcp --namespace=nada secret/metabase-google-sa -o json | jq -r '.data."meta_creds.json"' | base64 -d > test-metabase-sa.json)
.PHONY: test-sa

setup-metabase:
	./resources/scripts/configure_metabase.sh

local-with-auth: | env test-sa metabase-sa docker-build-metabase docker-compose-up setup-metabase
	@echo "Sourcing environment variables..."
	set -a && source ./.env && set +a && \
		STORAGE_EMULATOR_HOST=http://localhost:8082/storage/v1/ go run ./cmd/nada-backend --config ./config-local-online.yaml
.PHONY: local-with-auth

local: | env test-sa  setup-metabase
	@echo "Sourcing environment variables..."
	set -a && source ./.env && set +a && \
		GOOGLE_CLOUD_PROJECT=test STORAGE_EMULATOR_HOST=http://localhost:8082/storage/v1/ go run ./cmd/nada-backend --config ./config-local.yaml
.PHONY: local

local-deps: | docker-build-metabase-local-bq docker-build-apps docker-compose-up-fg
.PHONY: local-deps

docker-compose-up-fg:
	@echo "Starting dependencies with docker compose..."
	$(DOCKER_COMPOSE) up

docker-compose-up:
	@echo "Starting dependencies with docker compose..."
	$(DOCKER_COMPOSE ) up -d

migrate:
	go run github.com/pressly/goose/v3/cmd/goose -dir ./pkg/database/migrations postgres "user=nada-backend dbname=nada sslmode=disable password=postgres" up
.PHONY: migrate

generate-sql:
	cd pkg && $(GOBIN)/sqlc generate
.PHONY: generate-sql

generate: generate-sql
.PHONY: generate

linux-build:
	go build -a -installsuffix cgo -o $(APP) -ldflags "-s $(LDFLAGS)" ./cmd/nada-backend
.PHONY: linux-build

docker-build-metabase:
	@echo "Building metabase docker image..."
	docker image build -t metabase-nada-backend:latest -f Dockerfile-metabase-orig .

docker-build-metabase-local-bq:
	@echo "Building metabase docker image with local BigQuery..."
	docker image build -t metabase-nada-backend:latest -f Dockerfile-metabase-local .

docker-build-apps:
	docker image build -t nada-apps:latest -f Dockerfile-build .

docker-build:
	docker image build -t ghcr.io/navikt/$(APP):$(VERSION) -t ghcr.io/navikt/$(APP):latest .
.PHONY: docker-build

docker-push:
	docker image push ghcr.io/navikt/$(APP):$(VERSION)
	docker image push ghcr.io/navikt/$(APP):latest
.PHONY: docker-push

install-sqlc:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
.PHONY: install-sqlc
