# s3xplorer

[![GitHub release](https://img.shields.io/github/release/sgaunet/s3xplorer.svg)](https://github.com/sgaunet/s3xplorer/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/sgaunet/s3xplorer)](https://goreportcard.com/report/github.com/sgaunet/s3xplorer)
![GitHub Downloads](https://img.shields.io/github/downloads/sgaunet/s3xplorer/total)
![Coverage](https://raw.githubusercontent.com/wiki/sgaunet/s3xplorer/coverage-badge.svg)
[![coverage](https://github.com/sgaunet/s3xplorer/actions/workflows/coverage.yml/badge.svg)](https://github.com/sgaunet/s3xplorer/actions/workflows/coverage.yml)
[![Snapshot Build](https://github.com/sgaunet/s3xplorer/actions/workflows/snapshot.yml/badge.svg)](https://github.com/sgaunet/s3xplorer/actions/workflows/snapshot.yml)
[![Release Build](https://github.com/sgaunet/s3xplorer/actions/workflows/release.yml/badge.svg)](https://github.com/sgaunet/s3xplorer/actions/workflows/release.yml)
[![License](https://img.shields.io/github/license/sgaunet/s3xplorer.svg)](LICENSE)

s3xplorer is a web interface to parse a S3 bucket.

In the beginning, this project was a POC to play with AWS golang SDK v2 and the minio library. It's quite basic, it still needs refactor and improvements...

![Light mode](docs/light.png) ![Dark mode](docs/dark.png)

## Major Changes in v0.5.0

Starting with version 0.5.0, s3xplorer introduces a major architectural change to significantly improve performance:

### PostgreSQL Backend
- **Database-driven UI**: The web interface now reads from a PostgreSQL database instead of making direct S3 API calls
- **Background synchronization**: A background scanner periodically syncs S3 bucket contents to the database
- **Massive performance improvement**: Browsing large buckets is now instant, as data is served from the local database
- **Scheduled scanning**: Configure cron-based schedules for automatic bucket synchronization
- **Deletion tracking**: Optionally sync deletions to keep the database in sync with S3

### Migration Notes
- **Database required**: PostgreSQL is now required for the application to function
- **Configuration changes**: Update your configuration files to the new hierarchical format (see examples below)
- **First scan**: The initial scan may take time depending on bucket size, but subsequent access will be fast

See the configuration examples below for setting up the database and background scanning features.

## Install

* Use the binary in the release page
* Or the Docker image
* Or [helm chart](https://github.com/sgaunet/helm-s3xplorer)

Check next section to see how to configure it.

### Docker

```yml
version: '3.7'
services:
  minio-server:
    image: minio/minio:RELEASE.2024-11-07T00-52-20Z-cpuv1
    ports:
      - 9090:9000
      - 8080:8080
    environment: 
      - MINIO_ROOT_USER=minioadminn
      - MINIO_ROOT_PASSWORD=minioadminn
    volumes:
      - ./data:/export
    command: minio server /export --console-address 0.0.0.0:8080

  s3xplorer:
    image: sgaunet/s3xplorer:<version>
    ports:
      - 8081:8081
    volumes:
      - ./config.yaml:/cfg.yaml
    depends_on:
      - minio-server
```

Read configuration examples in [conf-examples](conf-examples) folder.

### Helm

* Here is the [documentation of the helm chart](https://github.com/sgaunet/helm-s3xplorer/blob/main/charts/s3xplorer/README.md).
* Here is the source code of the [helm chart](https://github.com/sgaunet/helm-s3xplorer).

```bash
helm repo add s3xplorer https://sgaunet.github.io/helm-s3xplorer/
helm repo update
helm search repo s3xplorer
```

## Configuration

Example with a local minio server:

```yaml
# S3 Configuration
s3:
  # set endpoint and region if SSO is not used
  endpoint: http://127.0.0.1:9090
  region: "us-east-1"
  # access_key and api_key are mandatory if SSO is not used
  access_key: minioadminn
  api_key: minioadminn
  sso_aws_profile: 
  bucket: example
  # set the prefix if you want to restrict the access to a specific folder
  # don't forget to add the trailing slash at the end
  prefix: rando/
  # Glacier restore settings
  restore_days: 1
  enable_glacier_restore: false
  skip_bucket_validation: false

# Database Configuration (optional - for PostgreSQL backend)
database:
  url: "postgres://postgres:postgres@localhost:5432/s3xplorer?sslmode=disable"

# Background Scanning Configuration (optional - requires database)
scan:
  enable_background_scan: true
  cron_schedule: "0 2 * * *"  # daily at 2 AM
  enable_initial_scan: false
  enable_deletion_sync: true

# Bucket Sync Configuration (optional)
bucket_sync:
  enable: true
  sync_threshold: "24h"
  delete_threshold: "168h"
  max_retries: 3

# Logging
# log_level: debug | info | warn | error
log_level: info
```

## Configuration with AWS SSO (not recommmended)

Example:

```yaml
# S3 Configuration
s3:
  endpoint:
  region: "eu-west-3"
  access_key: 
  api_key: 
  sso_aws_profile: dev
  bucket: my-bucket
  prefix: ""
  # Number of days that objects will be restored from Glacier (default: 2 if not specified)
  restore_days: 7
  # Whether to enable the Glacier restore button (default: false if not specified)
  # If set to false, the restore button will not be shown for archived objects
  enable_glacier_restore: true
  skip_bucket_validation: false

# Database Configuration (optional)
database:
  url: "postgres://postgres:postgres@localhost:5432/s3xplorer?sslmode=disable"

# Logging
# log_level: debug | info | warn | error
log_level: info
```

To use AWS SSO, you need to have the AWS CLI installed and configured with the SSO profile.

```bash
aws sso login --profile <profile>
s3xplorer -f config.yaml
```

Then you can use the profile in the configuration file.
It's not recommended to use AWS SSO with this tool because the session will expire and you will need to re-authenticate.

## Usage

```bash
s3xplorer -f config.yaml
```

With Docker:

```bash
docker run -v $(pwd)/config.yaml:/config.yaml -p 8080:8080 sgaunet/s3xplorer:latest -f /config.yaml
# or
docker run -v $(pwd)/config.yaml:/cfg.yaml -p 8081:8081 sgaunet/s3xplorer:latest
```

## Development

This project is using :

* Golang
* [Task for development](https://taskfile.dev/#/)
* Docker
* [Docker buildx](https://github.com/docker/buildx)
* Docker manifest
* [Goreleaser](https://goreleaser.com/)

### Tasks

Launch task to see all available tasks:

```bash
task
```

## Performance

Quite lighweight now since 0.3.0. Tests with 3 concurrents downloads of 5GB of each file, and less thant 30MB memory consumption.

## IAM Policy

A cloudformation stack is available in [docs/stack-cloudformation.yaml](docs/stack-cloudformation.yaml) to create a bucket and a user with the right policy to access the bucket.
