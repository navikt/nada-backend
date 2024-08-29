SHELL = bash
DATE = $(shell date "+%Y-%m-%d")
LAST_COMMIT = $(shell git --no-pager log -1 --pretty=%h)
VERSION ?= $(DATE)-$(LAST_COMMIT)
LDFLAGS := -X github.com/navikt/nada-backend/backend/version.Revision=$(shell git rev-parse --short HEAD) -X github.com/navikt/nada-backend/backend/version.Version=$(VERSION)

METABASE_VERSION := $(shell cat .metabase_version)
MOCKS_VERSION := v0.0.1

TARGET_ARCH := amd64
TARGET_OS   := linux

IMAGE_URL        := europe-north1-docker.pkg.dev
IMAGE_REPOSITORY := nada-prod-6977/nada-north

COMPOSE_DEPS_FULLY_LOCAL := db adminer gcs metabase-patched bq tk nc
COMPOS_DEPS_ONLINE_LOCAL := db adminer gcs metabase

APP = nada-backend

# A template function for installing binaries
define install-binary
	 @if ! command -v $(1) &> /dev/null; then \
		  echo "$(1) not found, installing..."; \
		  go install $(2); \
	 fi
endef

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

STATICCHECK          ?= $(shell command -v staticcheck || echo "$(GOBIN)/staticcheck")
STATICCHECK_VERSION  := v0.4.6
SQLC                 ?= $(shell command -v sqlc || echo "$(GOBIN)/sqlc")
SQLC_VERSION         := v1.27.0
GOFUMPT			     ?= $(shell command -v gofumpt || echo "$(GOBIN)/gofumpt")
GOFUMPT_VERSION	     := v0.6.0
GOLANGCILINT         ?= $(shell command -v golangci-lint || echo "$(GOBIN)/golangci-lint")
GOLANGCILINT_VERSION := v1.55.2

$(SQLC):
	$(call install-binary,sqlc,github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION))

$(STATICCHECK):
	$(call install-binary,staticcheck,honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION))

$(GOFUMPT):
	$(call install-binary,gofumpt,mvdan.cc/gofumpt@$(GOFUMPT_VERSION))

$(GOLANGCILINT):
	$(call install-binary,golangci-lint,github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCILINT_VERSION))

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

test: | pull-all
	METABASE_VERSION=$(METABASE_VERSION) CGO_ENABLED=1 CXX=clang++ CC=clang \
		CGO_CXXFLAGS=-Wno-everything CGO_LDFLAGS=-Wno-everything \
			go test -timeout 20m -race -coverprofile=coverage.txt -covermode=atomic -v ./...
.PHONY: test

staticcheck: $(STATICCHECK)
	$(STATICCHECK) ./...

gofumpt: $(GOFUMPT)
	$(GOFUMPT) -w .

lint: $(GOLANGCILINT)
	$(GOLANGCILINT) run
.PHONY: lint

check: | gofumpt lint staticcheck test
.PHONY: check

compile: $(RELEASE_DIR)
	@echo "Compiling cmd applications..."
	@CGO_ENABLED=1 CXX=clang++ CC=clang $(GO) mod tidy
	@for d in cmd/*; do \
		app=$$(basename $$d); \
		echo "Compiling $$app..."; \
		CGO_ENABLED=1 CXX=clang++ CC=clang CGO_CXXFLAGS=-Wno-everything CGO_LDFLAGS=-Wno-everything $(GO) build -o $(RELEASE_DIR)/$$app ./$$d; \
	done
	@echo "Compile complete. Binaries are located in $(RELEASE_DIR)"
.PHONY: compile

generate: $(SQLC)
	cd pkg && $(SQLC) generate
.PHONY: generate

release:
	GOOS=linux GOARCH=amd64 CGO_EMABLED=0 $(GO) build -o $(APP) \
		-ldflags '-linkmode "external" -extldflags "-static" -w -s $(LDFLAGS)' ./cmd/nada-backend/main.go
.PHONY: release

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
.PHONY: metabase-sa

setup-metabase:
	./resources/scripts/configure_metabase.sh
.PHONY: setup-metabase

run-online: | env test-sa metabase-sa docker-build-metabase docker-compose-up setup-metabase
	@echo "Sourcing environment variables..."
	set -a && source ./.env && set +a && \
		STORAGE_EMULATOR_HOST=http://localhost:8082/storage/v1/ $(GO) run ./cmd/nada-backend --config ./config-local-online.yaml
.PHONY: run-online

start-run-online-deps: | docker-login pull-all
	@echo "Starting dependencies with docker compose... (online)"
	@echo "Mocks version: $(MOCKS_VERSION)"
	@echo "Metabase version: $(METABASE_VERSION)"
	MOCKS_VERSION=$(MOCKS_VERSION) METABASE_VERSION=$(METABASE_VERSION) $(DOCKER_COMPOSE ) up -d $(COMPOS_DEPS_ONLINE_LOCAL)
.PHONY: start-run-online-deps

run: | start-run-deps env test-sa setup-metabase
	@echo "Sourcing environment variables..."
	set -a && source ./.env && set +a && \
		GOOGLE_CLOUD_PROJECT=test STORAGE_EMULATOR_HOST=http://localhost:8082/storage/v1/ $(GO) run ./cmd/nada-backend --config ./config-local.yaml
.PHONY: run

start-run-deps: | docker-login pull-all
	@echo "Starting dependencies with docker compose... (fully local)"
	@echo "Mocks version: $(MOCKS_VERSION)"
	@echo "Metabase version: $(METABASE_VERSION)"
	MOCKS_VERSION=$(MOCKS_VERSION) METABASE_VERSION=$(METABASE_VERSION) $(DOCKER_COMPOSE) up -d $(COMPOSE_DEPS_FULLY_LOCAL)
.PHONY: start-run-deps

docker-login:
	@echo "Logging in to Google Cloud..."
	gcloud auth configure-docker $(IMAGE_URL)
.PHONY: docker-login

build-push-all: | build-all push-all
.PHONY: build-push-all

pull-all: | pull-metabase pull-metabase-patched pull-deps
.PHONY: pull-all

pull-metabase:
	@echo "Pulling metabase docker image from registry..."
	docker pull $(IMAGE_URL)/$(IMAGE_REPOSITORY)/metabase:$(METABASE_VERSION)
.PHONY: pull-metabase

pull-metabase-patched:
	@echo "Pulling patched metabase docker image from registry..."
	docker pull $(IMAGE_URL)/$(IMAGE_REPOSITORY)/metabase-patched:$(METABASE_VERSION)
.PHONY: pull-metabase-patched

pull-deps:
	@echo "Pulling nada-backend mocks docker image from registry..."
	docker pull $(IMAGE_URL)/$(IMAGE_REPOSITORY)/nada-backend-mocks:$(MOCKS_VERSION)
.PHONY: pull-deps

build-all: | build-metabase build-metabase-patched build-deps
.PHONY: build-all

build-metabase:
	@echo "Building original metabase docker image, for version: $(METABASE_VERSION)"
	docker image build --platform $(TARGET_OS)/$(TARGET_ARCH) --tag $(IMAGE_URL)/$(IMAGE_REPOSITORY)/metabase:$(METABASE_VERSION) \
		--build-arg METABASE_VERSION=$(METABASE_VERSION) --file resources/images/metabase/Dockerfile .
.PHONY: build-metabase

build-metabase-patched:
	@echo "Building patched metabase docker image, for version: $(METABASE_VERSION)"
	docker image build --platform $(TARGET_OS)/$(TARGET_ARCH) --tag $(IMAGE_URL)/$(IMAGE_REPOSITORY)/metabase-patched:$(METABASE_VERSION) \
		--build-arg METABASE_VERSION=$(METABASE_VERSION) --file resources/images/metabase/Dockerfile-bq-patch .
.PHONY: build-metabase-patched

build-deps: build-metabase-patched
	@echo "Building nada-backend mocks..."
	docker image build --platform $(TARGET_OS)/$(TARGET_ARCH) --tag $(IMAGE_URL)/$(IMAGE_REPOSITORY)/nada-backend-mocks:$(MOCKS_VERSION) \
		--file resources/images/nada-backend/Dockerfile-mocks .
.PHONY: build-deps

push-all: | push-metabase push-metabase-patched push-deps
.PHONY: push-all

push-metabase:
	@echo "Pushing metabase docker image to registry..."
	docker push $(IMAGE_URL)/$(IMAGE_REPOSITORY)/metabase:$(METABASE_VERSION)
.PHONY: push-metabase

push-metabase-patched:
	@echo "Pushing patched metabase docker image to registry..."
	docker push $(IMAGE_URL)/$(IMAGE_REPOSITORY)/metabase-patched:$(METABASE_VERSION)
.PHONY: push-metabase-patched

push-deps:
	@echo "Pushing nada-backend mocks docker image to registry..."
	docker push $(IMAGE_URL)/$(IMAGE_REPOSITORY)/nada-backend-mocks:$(MOCKS_VERSION)
.PHONY: push-deps

check-images:
	@./resources/scripts/check_images.sh $(IMAGE_URL)/$(IMAGE_REPOSITORY) metabase:$(METABASE_VERSION) metabase-patched:$(METABASE_VERSION) nada-backend-mocks:$(MOCKS_VERSION)
.PHONY: check-images
