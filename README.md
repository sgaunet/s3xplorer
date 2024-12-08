[![GitHub release](https://img.shields.io/github/release/sgaunet/s3xplorer.svg)](https://github.com/sgaunet/s3xplorer/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/sgaunet/s3xplorer)](https://goreportcard.com/report/github.com/sgaunet/s3xplorer)
![GitHub Downloads](https://img.shields.io/github/downloads/sgaunet/s3xplorer/total)
[![License](https://img.shields.io/github/license/sgaunet/s3xplorer.svg)](LICENSE)

# s3xplorer

s3xplorer is a web interface to parse a S3 bucket.

In the beginning, this project was a POC to play with aws golang sdk v2 and the minio library. It's quite basic, it still needs refactor and improvements...

![s3xplorer](img/v0.1.0.png)

## Install

* Use the binary in the release page
* Or the Docker image
* An helm chart will be available soon

## Configuration

Example with a local minio server:

```yaml
s3endpoint: http://127.0.0.1:9090
s3region: "us-east-1"
accesskey: minioadminn
apikey: minioadminn
awsssoprofile: 
bucket: example

loglevel: info
```

## Configuration with AWS SSO (not recommmended)

Example:

```yaml
s3endpoint:
s3region: "eu-west-3"
accesskey: 
apikey: 
ssoawsprofile: dev
bucket: my-bucket

loglevel: info
```

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
