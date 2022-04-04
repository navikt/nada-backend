.PHONY: test integration-test local-with-auth local linux-build docker-build docker-push run-postgres-test stop-postgres-test install-sqlc 
DATE = $(shell date "+%Y-%m-%d")
LAST_COMMIT = $(shell git --no-pager log -1 --pretty=%h)
VERSION ?= $(DATE)-$(LAST_COMMIT)
LDFLAGS := -X github.com/navikt/nada-backend/backend/version.Revision=$(shell git rev-parse --short HEAD) -X github.com/navikt/nada-backend/backend/version.Version=$(VERSION)
APP = nada-backend
SQLC_VERSION ?= "v1.10.0"
# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
	GOBIN=$(shell go env GOPATH)/bin
else
	GOBIN=$(shell go env GOBIN)
endif

-include .env

test:
	go test ./... -count=1

integration-test:
	go test ./... -count=1 -tags=integration_test

env:
	echo "NADA_CLIENT_ID=$(shell kubectl get --context=dev-gcp --namespace=nada secret/google-oauth -o jsonpath='{.data.CLIENT_ID}' | base64 -d)" > .env
	echo "NADA_CLIENT_SECRET=$(shell kubectl get --context=dev-gcp --namespace=nada secret/google-oauth -o jsonpath='{.data.CLIENT_SECRET}' | base64 -d)" >> .env
	echo "GITHUB_READ_TOKEN=$(shell kubectl get secret --context=dev-gcp --namespace=nada github-read-token -o jsonpath='{.data.GITHUB_READ_TOKEN}' | base64 -d)" >> .env
	echo "METABASE_USERNAME=$(shell kubectl get secret --context=dev-gcp --namespace=nada metabase-sa -o jsonpath='{.data.METABASE_USERNAME}' | base64 -d)" >> .env
	echo "METABASE_PASSWORD=$(shell kubectl get secret --context=dev-gcp --namespace=nada metabase-sa -o jsonpath='{.data.METABASE_PASSWORD}' | base64 -d)" >> .env

test-sa:
	$(shell kubectl get --context=dev-gcp --namespace=nada secret/google-credentials -o json | jq -r '.data."sa.json"' | base64 -d > test-sa.json)

local-with-auth:
	go run ./cmd/nada-backend \
	--oauth2-client-id=$(NADA_CLIENT_ID) \
	--oauth2-client-secret=$(NADA_CLIENT_SECRET) \
	--teams-token=$(GITHUB_READ_TOKEN) \
	--bind-address=127.0.0.1:8080 \
	--hostname=localhost \
	--service-account-file=./test-sa.json \
	--google-admin-subject=johnny.horvi@nav.no \
	--metabase-username=$(METABASE_USERNAME) \
	--metabase-password=$(METABASE_PASSWORD) \
	--teamkatalogen-url=https://teamkatalog-api.intern.nav.no \
	--extract-bucket=nada-csv-export-dev \
	--log-level=debug

local:
	go run ./cmd/nada-backend \
	--teams-token=$(GITHUB_READ_TOKEN) \
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

install-sqlc:
	go install github.com/kyleconroy/sqlc/cmd/sqlc@$(SQLC_VERSION)
