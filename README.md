# Data management API for NAV

It serves a REST-API for managing data products, and provides functionality for self-service access to the data source.

## Getting started with local development

1. Install required dependencies

- https://clang.llvm.org
- https://docs.docker.com/engine/install/
- https://cloud.google.com/sdk/gcloud
- https://kubernetes.io/docs/tasks/tools/#kubectl

2. Configure `gcloud` so you can [access Nais clusters](https://doc.nais.io/operate/how-to/command-line-access/#google-cloud-platform-gcp)  
3. Login to GCP and configure docker
```bash
gcloud auth login --update-adc
gcloud auth configure-docker europe-north1-docker.pkg.dev 

# There also exists a make target for login to docker:
make docker-login
```
4. (Optional) If you are on mac with arm (m1, m2, m3, etc.) install rosetta
```bash
softwareupdate --install-rosetta
```
5. Run som build commands

```bash
# Build all binaries
make build

# Run the tests
make test
```

## Run with fully local resources

With this configuration all dependencies run as containers, as can be seen in `docker-compose.yaml`:
- Google BigQuery using [bigquery-emulator](https://github.com/goccy/bigquery-emulator), with additional mocks for the 
  IAM Policy 
  endpoints
- Google Cloud Storage using [fake-gcs-server](https://github.com/fsouza/fake-gcs-server)
- [Metabase](https://github.com/metabase/metabase) with a [patch](resources/images/metabase/001-bigquery-cloud-sdk-no-auth.patch) for enabling use of bigquery-emulator
- Fake API servers for `teamkatalogen` and `naisconsole`

There are still a couple of services missing, though much functionality should work without this:
- Fetching of Google Groups
- Creating Google Cloud Service Accounts

1. Start the dependencies and API
```bash
# Starts the dependencies in the background, and runs the API in the foreground
$ make run
```
2. (Optional): Start the [nada-frontend](https://github.com/navikt/nada-frontend/?tab=readme-ov-file#development)

3. (Optional): Take a look at the [locally running Metabase](http://localhost:8083), the username is: `nada@nav.no`,
   and password is: `superdupersecret1`

## Making changes to the database or generated models and queries

1. [Migrations](pkg/database/migrations) allows you to modify the existing database, these are automatically applied during startup of the application
2. [Queries](pkg/database/queries) lets you generate new models and queries based on the existing structure

**NB:** If you make changes to the *Queries* remember to run the generate command so your changes are propagated:

```bash
$ make generate
```

## Bumping the Metabase version
The file [.metabase_version](.metabase_version) controls the version of [Metabase](https://metabase.com) that is 
used in tests and for deployment to **dev** and **prod**. Check the Metabase [releases](https://github.com/metabase/metabase/releases) page 
for available versions; we follow the Metabase Enterprise track.

When you bump this version the following events will occur when you make a PR:

1. We build two Metabase images, which are used during integration tests and for local development
- metabase: un-modified version of Metabase when running nada-backend locally towards GCP services
- metabase-patched: modified version of Metabase that allows us to connect to bigquery-emulator running on the host
2. We run the nada-backend integration tests using the new version of Metabase
3. We deploy the new version of Metabase to `dev`

On merge to `main`:

1. We deploy the new version of Metabase `prod`

## Bumping the Mocks version
In the [Makefile](Makefile) we set the target version for the mocks. If you change the mocks, you also need to bump 
the `MOCKS_VERSION`, so we get the latest changes.

## Update the images locally

We build and push images for the patched metabase and customized big-query emulator to speed up local development and integration tests. If you need to make changes to these: 

1. Make changes to the [base images](resources/images)

**Note:** building the big query emulator requires quite a bit of memory, so if you see something like `clang++:
signal: killed` you need to increase the amount of memory you allocate to your container run-time.

2. Build the new images locally
```bash
$ make build-all
```
3. (optional) Push the images to the container registry; requires that you have run `make docker-login`
```
$ make push-all
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
