SERVICE_SHORT_NAME=account-api
SERVICE_NAME=dl-flow/account-api
SERVICE_DIR=cmd/account-api

# Build information
#----------------------------------------------------------------------
REPO:=gcr.io
COMMIT_SHA:=$(shell git rev-parse --short HEAD)
IMAGE_URL:=${REPO}/${SERVICE_NAME}

IMAGE_WITH_COMMIT=${IMAGE_URL}:${COMMIT_SHA}
IMAGE_WITH_LATEST:=${IMAGE_URL}:latest

K8S_YAMLS_LOCATION_TESTNET := ./k8s/staging/testnet
K8S_YAMLS_LOCATION_MAINNET := ./k8s/prod/mainnet

export GO111MODULE := on
export GOARCH := amd64
export GOOS := linux
export DOCKER_BUILDKIT := 1
export COMPOSE_DOCKER_CLI_BUILD := 1

.PHONY: run
run:
	docker-compose up --build --renew-anon-volumes

.PHONY: run-with-local-emulator
run-with-local-emulator:
	docker-compose --f docker-compose.local-emulator.yml up --build

.PHONY: docker-build
docker-build:
	docker build \
		-f ${SERVICE_DIR}/Dockerfile \
		-t ${IMAGE_WITH_COMMIT} \
		-t ${IMAGE_WITH_LATEST} \
		.

.PHONY: docker-push
docker-push:
	@docker push ${IMAGE_WITH_COMMIT}
	@docker push ${IMAGE_WITH_LATEST}
