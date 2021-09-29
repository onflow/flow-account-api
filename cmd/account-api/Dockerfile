# syntax = docker/dockerfile:experimental

## (1) Build the app binary
FROM golang:1.16 AS build-app

# Build the app binary in /app
RUN mkdir /app
WORKDIR /app

COPY ./cmd/account-api/migrations /migrations
COPY . .

RUN  GO111MODULE=on GOOS=linux GOARCH=amd64 go build -o ./app ./cmd/account-api

RUN chmod a+x /app/app

## (2) Add the statically linked binary to a distroless image
FROM gcr.io/distroless/base

COPY --from=build-app /app/app /bin/app
COPY --from=build-app /migrations /data/migrations/

ENTRYPOINT ["/bin/app"]
