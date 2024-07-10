# Data management API for NAV

It serves a REST-API for managing data products, and provides functionality for self-service access to the data source.

## Getting started with local development

1. Install required dependencies

- https://clang.llvm.org
- https://docs.docker.com/engine/install/
- https://cloud.google.com/sdk/gcloud
- https://kubernetes.io/docs/tasks/tools/#kubectl

2. Configure `gcloud` so you can [access Nais clusters](https://doc.nais.io/operate/how-to/command-line-access/#google-cloud-platform-gcp)  
3. Login to GCP
```bash
gcloud auth login --update-adc
```
4. Run som build commands

```bash
# Build all binaries
$ make build

# Run the tests
$ make test
```

## Run with fully local resources

With this configuration all dependencies run as containers, as can be seen in `docker-compose.yaml`:
- Google BigQuery using [bigquery-emulator](https://github.com/goccy/bigquery-emulator), with additional mocks for the 
  IAM Policy 
  endpoints
- Google Cloud Storage using [fake-gcs-server](https://github.com/fsouza/fake-gcs-server)
- [Metabase](https://github.com/metabase/metabase) with a [patch](resources/metabase/001-bigquery-cloud-sdk-no-auth.patch) for enabling use of bigquery-emulator
- Fake API servers for `teamkatalogen` and `naisconsole`

1. Start the dependencies
```bash
# Builds and runs dependencies in the foreground
$ make local-deps 
```

2. Start the backend
```bash
 # Builds and runs nada-backend using config-local.yaml
$ make local
```

**Note:** building the big query emulator requires quite a bit of memory, so if you see something like `clang++:
signal: killed` you need to increase the amount of memory you allocate to your container run-time.

3. (Optional): Start the [nada-frontend](https://github.com/navikt/nada-frontend/?tab=readme-ov-file#development)

4. (Optional): Take a look at the [locally running Metabase](http://localhost:8083), the username is: `nada@nav.no`,
   and password is: `superdupersecret1`

## Deployment
The application needs two GCP service accounts which are mounted in at runtime from two secrets in Google Secret Manager. These are:

- `nada-backend-google-credentials`
- `nada-backend-metabase-sa`

It also needs a GCP project where the BigQuery API is enabled to host keys for pseudonymization. 

The following values used by the application are also stored in Google Secret Manager in the secret `nada-backend`:

```
CLIENT_ID
CLIENT_SECRET
GITHUB_READ_TOKEN
GOOGLE_ADMIN_IMPERSONATION_SUBJECT
METABASE_PASSWORD
METABASE_USERNAME
SLACK_URL
```

## Architecture

```mermaid
flowchart TD
    %% Define the layers
    Transport["Transport (e.g., HTTP)"] --> Router["Router (METHOD /path)"]
    Router --> Endpoint["Encoding and decoding (JSON)"]
    Endpoint --> Handler["Handler (e.g., Request Handlers)"]
    Handler --> Service1["Service1 (e.g., Data Processing Service)"]
    Handler --> Service2["Service2 (e.g., Authentication Service)"]
    Handler --> ServiceN["ServiceN"]
    Service1 --> Model1["Model1 (e.g., Big Query Model)"]
    Service2 --> Model2["Model2 (e.g., Data accesss)"]
    ServiceN --> ModelN["ModelN (e.g., Metabase)"]
    Service1 --> Storage1["Storage1 (e.g., PostgreSQL)"]
    Service2 --> Storage2["Storage2 (e.g., MongoDB)"]
    Service2 --> StorageN["StorageN"]
    Service1 --> API1["External API 1 (e.g., GCP Big Query API)"]
    Service2 --> API2["External API 2 (e.g., Metabase API)"]
    ServiceN --> APIN

%% Styling classes
classDef service fill:#f9f,stroke:#333,stroke-width:2px;
class Service1,Service2,ServiceN service;
classDef model fill:#bbf,stroke:#333,stroke-width:2px;
class Model1,Model2,ModelN model;
classDef storage fill:#ffb,stroke:#333,stroke-width:2px;
class Storage1,Storage2,StorageN storage;
classDef api fill:#bfb,stroke:#333,stroke-width:2px;
class API1,API2,APIN api;
```
