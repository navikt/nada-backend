.PHONY: test integration-test local-with-auth local linux-build docker-build docker-push run-local-firestore
DATE = $(shell date "+%Y-%m-%d")
LAST_COMMIT = $(shell git --no-pager log -1 --pretty=%h)
VERSION ?= $(DATE)-$(LAST_COMMIT)
LDFLAGS := -X github.com/navikt/datakatalogen/backend/version.Revision=$(shell git rev-parse --short HEAD) -X github.com/navikt/datakatalogen/backend/version.Version=$(VERSION)
APP = datakatalogen

test:
	go test ./... -count=1

integration-test:
	./hack/with_firebase_emulator.sh make test

run-local-firestore:
	gcloud beta emulators firestore start --host-port=localhost:6969

local-with-auth:
	go run cmd/backend/main.go \
	--teams-url=https://raw.githubusercontent.com/navikt/teams/main/teams.json \
	--oauth2-client-secret=$(shell kubectl get --context=dev-gcp --namespace=aura `kubectl get secret --context=dev-gcp --namespace=aura --sort-by='{.metadata.creationTimestamp}' -l app=datakatalogen,type=azurerator.nais.io -o name | tail -1` -o jsonpath='{.data.AZURE_APP_CLIENT_SECRET}' | base64 -d) \
	--teams-token=$(shell kubectl get secret --context=dev-gcp --namespace=aura github-read-token -o jsonpath='{.data.GITHUB_READ_TOKEN}' | base64 -d) \
	--oauth2-client-id=$(shell kubectl get --context=dev-gcp --namespace=aura `kubectl get secret --context=dev-gcp --namespace=aura --sort-by='{.metadata.creationTimestamp}' -l app=datakatalogen,type=azurerator.nais.io -o name | tail -1` -o jsonpath='{.data.AZURE_APP_CLIENT_ID}' | base64 -d) \
	--oauth2-tenant-id=62366534-1ec3-4962-8869-9b5535279d0b \
	--bind-address=127.0.0.1:8080 \
	--firestore-google-project-id=aura-dev-d9f5 \
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
	--firestore-google-project-id=aura-dev-d9f5 \
	--dataproducts-collection=new-access-format \
	--access-updates-collection=access-updates \
	--hostname=localhost \
	--log-level=debug \
	--state=$(shell gcloud secrets versions access --secret datakatalogen-state latest --project aura-dev-d9f5 | cut -d= -f2)

linux-build:
	go build -a -installsuffix cgo -o $(APP) -ldflags "-s $(LDFLAGS)" cmd/backend/main.go

docker-build:
	docker image build -t ghcr.io/navikt/$(APP):$(VERSION) -t ghcr.io/navikt/$(APP):latest .

docker-push:
	docker image push ghcr.io/navikt/$(APP):$(VERSION)
	docker image push ghcr.io/navikt/$(APP):latest

