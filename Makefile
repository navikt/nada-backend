DATE = $(shell date "+%Y-%m-%d")
LAST_COMMIT = $(shell git --no-pager log -1 --pretty=%h)
VERSION ?= $(DATE)-$(LAST_COMMIT)
LDFLAGS := -X github.com/navikt/nada-backend/backend/version.Revision=$(shell git rev-parse --short HEAD) -X github.com/navikt/nada-backend/backend/version.Version=$(VERSION)
APP = nada-backend
SQLC_VERSION ?= "v1.23.0"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
	GOBIN=$(shell go env GOPATH)/bin
else
	GOBIN=$(shell go env GOBIN)
endif

-include .env

test:
	go test ./... -count=1
.PHONY: test

integration-test:
	go test ./... -count=1 -tags=integration_test
.PHONY: integration-test

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

local-with-auth: | env test-sa metabase-sa docker-build-metabase
	@echo "Sourcing environment variables..."
	set -a && source ./.env && set +a && \
		STORAGE_EMULATOR_HOST=http://localhost:8082/storage/v1/ go run ./cmd/nada-backend --config ./config-local-online.yaml
.PHONY: local-with-auth

docker-compose-up:
	@echo "Starting dependencies with docker-compose..."
	docker-compose up -d

local:
	@echo "Sourcing environment variables and starting nada-backend..."
	set -a && source ./.env && set +a && \
		STORAGE_EMULATOR_HOST=http://localhost:8082/storage/v1/ go run ./cmd/nada-backend --config ./config-local.yaml
.PHONY: local

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
	docker image build -t metabase-nada-backend:latest -f Dockerfile.metabase .

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
