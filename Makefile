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

test:
	go test ./... -count=1

integration-test: stop-postgres-test run-postgres-test run-integration-test stop-postgres-test

run-postgres-test:
	docker run -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=nada --rm --name postgres-test -p 5433:5432 -d postgres:12

stop-postgres-test:
	docker stop postgres-test || echo "okidoki"

run-integration-test:
	go test ./... -count=1 -tags=integration_test

local-with-auth:
	go run cmd/backend/main.go \
	--teams-url=https://raw.githubusercontent.com/navikt/teams/main/teams.json \
	--oauth2-client-secret=$(shell kubectl get --context=dev-gcp --namespace=aura `kubectl get secret --context=dev-gcp --namespace=aura --sort-by='{.metadata.creationTimestamp}' -l app=datakatalogen,type=azurerator.nais.io -o name | tail -1` -o jsonpath='{.data.AZURE_APP_CLIENT_SECRET}' | base64 -d) \
	--teams-token=$(shell kubectl get secret --context=dev-gcp --namespace=aura github-read-token -o jsonpath='{.data.GITHUB_READ_TOKEN}' | base64 -d) \
	--oauth2-client-id=$(shell kubectl get --context=dev-gcp --namespace=aura `kubectl get secret --context=dev-gcp --namespace=aura --sort-by='{.metadata.creationTimestamp}' -l app=datakatalogen,type=azurerator.nais.io -o name | tail -1` -o jsonpath='{.data.AZURE_APP_CLIENT_ID}' | base64 -d) \
	--oauth2-tenant-id=62366534-1ec3-4962-8869-9b5535279d0b \
	--bind-address=127.0.0.1:8080 \
	--dataproducts-collection=new-access-format \
	--access-updates-collection=access-updates \
	--hostname=localhost \
	--log-level=debug \
	--state=$(shell gcloud secrets versions access --secret datakatalogen-state latest --project aura-dev-d9f5 | cut -d= -f2)

local:
	go run cmd/backend/main.go \
	--teams-url=https://raw.githubusercontent.com/navikt/teams/main/teams.json \
	--teams-token=$(shell gcloud secrets versions access --secret github-read-token latest --project aura-dev-d9f5 | cut -d= -f2) \
	--development-mode=true \
	--bind-address=127.0.0.1:8080 \
	--dataproducts-collection=new-access-format \
	--access-updates-collection=access-updates \
	--hostname=localhost \
	--log-level=debug \
	--state=$(shell gcloud secrets versions access --secret datakatalogen-state latest --project aura-dev-d9f5 | cut -d= -f2)

migrate:
	go run github.com/pressly/goose/v3/cmd/goose -dir ./pkg/database/migrations postgres "user=postgres dbname=nada sslmode=disable password=navikt" up

generate: 
	cd pkg && $(GOBIN)/sqlc generate
	mkdir -p pkg/openapi
	go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen -package openapi --generate "types,chi-server,spec" ./spec-v1.0.yaml > ./pkg/openapi/nada.gen.go

linux-build:
	go build -a -installsuffix cgo -o $(APP) -ldflags "-s $(LDFLAGS)" cmd/nada-backend/main.go

docker-build:
	docker image build -t ghcr.io/navikt/$(APP):$(VERSION) -t ghcr.io/navikt/$(APP):latest .

docker-push:
	docker image push ghcr.io/navikt/$(APP):$(VERSION)
	docker image push ghcr.io/navikt/$(APP):latest

install-sqlc:
	go install github.com/kyleconroy/sqlc/cmd/sqlc@$(SQLC_VERSION)
