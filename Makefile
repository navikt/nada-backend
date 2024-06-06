.PHONY: test integration-test local-with-auth local linux-build docker-build docker-push run-postgres-test stop-postgres-test install-sqlc 
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

integration-test:
	go test ./... -count=1 -tags=integration_test

env:
	echo "NADA_CLIENT_ID=$(shell kubectl get --context=dev-gcp --namespace=nada `kubectl get secret --context=dev-gcp --namespace=nada --sort-by='{.metadata.creationTimestamp}' -l app=nada-backend,type=azurerator.nais.io -o name | tail -1` -o jsonpath='{.data.AZURE_APP_CLIENT_ID}' | base64 -d)" > .env
	echo "NADA_CLIENT_SECRET=$(shell kubectl get --context=dev-gcp --namespace=nada `kubectl get secret --context=dev-gcp --namespace=nada --sort-by='{.metadata.creationTimestamp}' -l app=nada-backend,type=azurerator.nais.io -o name | tail -1` -o jsonpath='{.data.AZURE_APP_CLIENT_SECRET}' | base64 -d)" >> .env
	echo "NADA_CLIENT_TENANT=$(shell kubectl get --context=dev-gcp --namespace=nada `kubectl get secret --context=dev-gcp --namespace=nada --sort-by='{.metadata.creationTimestamp}' -l app=nada-backend,type=azurerator.nais.io -o name | tail -1` -o jsonpath='{.data.AZURE_APP_TENANT_ID}' | base64 -d)" >> .env
	echo "GITHUB_READ_TOKEN=$(shell kubectl get secret --context=dev-gcp --namespace=nada github-read-token -o jsonpath='{.data.GITHUB_READ_TOKEN}' | base64 -d)" >> .env
	echo "METABASE_USERNAME=$(shell kubectl get secret --context=dev-gcp --namespace=nada nada-backend -o jsonpath='{.data.METABASE_USERNAME}' | base64 -d)" >> .env
	echo "METABASE_PASSWORD=$(shell kubectl get secret --context=dev-gcp --namespace=nada nada-backend -o jsonpath='{.data.METABASE_PASSWORD}' | base64 -d)" >> .env
	echo "CONSOLE_API_KEY=$(shell kubectl get secret --context=dev-gcp --namespace=nada nada-backend -o jsonpath='{.data.CONSOLE_API_KEY}' | base64 -d)" >> .env
	echo "AMPLITUDE_API_KEY=$(shell kubectl get secret --context=dev-gcp --namespace=nada nada-backend -o jsonpath='{.data.AMPLITUDE_API_KEY}' | base64 -d)" >> .env

test-sa:
	$(shell kubectl get --context=dev-gcp --namespace=nada secret/nada-backend-google-credentials -o json | jq -r '.data."sa.json"' | base64 -d > test-sa.json)

local-with-auth:
	STORAGE_EMULATOR_HOST=http://localhost:8082/storage/v1/ GCP_STORY_BUCKET_NAME=nada-quarto-storage-dev DASHBOARD_PA_ID=6dbeedea-b23e-4bf7-a1cb-21d02d15e452 go run ./cmd/nada-backend \
	--oauth2-client-id=$(NADA_CLIENT_ID) \
	--oauth2-client-secret=$(NADA_CLIENT_SECRET) \
	--oauth2-tenant-id=$(NADA_CLIENT_TENANT) \
	--teams-token=$(GITHUB_READ_TOKEN) \
	--bind-address=127.0.0.1:8080 \
	--hostname=localhost \
	--service-account-file=./test-sa.json \
	--google-admin-subject=johnny.horvi@nav.no \
	--metabase-username=$(METABASE_USERNAME) \
	--metabase-password=$(METABASE_PASSWORD) \
	--amplitude-api-key=$(AMPLITUDE_API_KEY) \
	--teamkatalogen-url=https://teamkatalog-api.intern.nav.no \
	--polly-url=https://polly.intern.dev.nav.no/process \
	--team-projects-url=https://raw.githubusercontent.com/nais/teams/master/gcp-projects/dev-output.json \
	--story-bucket=nada-quarto-storage-dev \
	--console-api-key="$(CONSOLE_API_KEY)" \
	--nada-token-creds=1234 \
	--log-level=debug \
	--central-data-project=datamarkedsplassen-dev \ 


local:
	STORAGE_EMULATOR_HOST=http://localhost:8082/storage/v1/ GCP_STORY_BUCKET_NAME=nada-quarto-storage-dev DASHBOARD_PA_ID=Mocked-001 go run ./cmd/nada-backend \
	--bind-address=127.0.0.1:8080 \
	--hostname=localhost \
	--mock-auth \
	--skip-metadata-sync \
	--story-bucket=nada-quarto-storage-dev \
	--log-level=debug \
	--nada-token-creds=1234 \
	--slack-token=$(SLACK_TOKEN)

migrate:
	go run github.com/pressly/goose/v3/cmd/goose -dir ./pkg/database/migrations postgres "user=postgres dbname=nada sslmode=disable password=postgres" up

generate-sql:
	cd pkg && $(GOBIN)/sqlc generate

generate: generate-sql

linux-build:
	go build -a -installsuffix cgo -o $(APP) -ldflags "-s $(LDFLAGS)" ./cmd/nada-backend

docker-build:
	docker image build -t ghcr.io/navikt/$(APP):$(VERSION) -t ghcr.io/navikt/$(APP):latest .

docker-push:
	docker image push ghcr.io/navikt/$(APP):$(VERSION)
	docker image push ghcr.io/navikt/$(APP):latest

install-sqlc:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
