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
	--teams-url=https://raw.githubusercontent.com/navikt/teams/main/teams.json \
	--oauth2-client-secret=$(shell kubectl get --context=dev-gcp --namespace=dataplattform `kubectl get secret --context=dev-gcp --namespace=dataplattform --sort-by='{.metadata.creationTimestamp}' -l app=nada-backend,type=azurerator.nais.io -o name | tail -1` -o jsonpath='{.data.AZURE_APP_CLIENT_SECRET}' | base64 -d) \
	--teams-token=$(shell kubectl get secret --context=dev-gcp --namespace=dataplattform github-read-token -o jsonpath='{.data.GITHUB_READ_TOKEN}' | base64 -d) \
	--oauth2-client-id=$(shell kubectl get --context=dev-gcp --namespace=dataplattform `kubectl get secret --context=dev-gcp --namespace=dataplattform --sort-by='{.metadata.creationTimestamp}' -l app=nada-backend,type=azurerator.nais.io -o name | tail -1` -o jsonpath='{.data.AZURE_APP_CLIENT_ID}' | base64 -d) \
	--oauth2-tenant-id=62366534-1ec3-4962-8869-9b5535279d0b \
	--bind-address=127.0.0.1:8080 \
	--hostname=localhost \
	--log-level=debug

local:
	go run ./cmd/nada-backend \
	--teams-url=https://raw.githubusercontent.com/navikt/teams/main/teams.json \
	--teams-token=$(shell kubectl get secret --context=dev-gcp --namespace=dataplattform github-read-token -o jsonpath='{.data.GITHUB_READ_TOKEN}' | base64 -d) \
	--bind-address=127.0.0.1:8080 \
	--hostname=localhost \
	--mock-auth \
	--log-level=debug

migrate:
	go run github.com/pressly/goose/v3/cmd/goose -dir ./pkg/database/migrations postgres "user=postgres dbname=nada sslmode=disable password=postgres" up

generate: 
	cd pkg && $(GOBIN)/sqlc generate
	mkdir -p pkg/openapi
	$(GOBIN)/oapi-codegen -package openapi --generate "types,chi-server,spec" ./spec-v1.0.yaml > ./pkg/openapi/nada.gen.go

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
