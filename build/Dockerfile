# syntax=docker/dockerfile:1
FROM golang:1.27.1-alpine3.24@sha256:cf6fca6641884b8433441b2b0652976f975e1d0fdd26d177eaaf8596087f3125 AS build

ENV CGO_ENABLED=0

WORKDIR /go/src/app

COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum

RUN go mod download

COPY ./cmd ./cmd
COPY ./internal ./internal
COPY ./main.go ./main.go

RUN go build -v \
      -o /out/main \
      -ldflags "-extldflags '-static'" \
      ./main.go

FROM alpine:3.24.1@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

RUN set -eux \
  && addgroup \
      --gid 1000 \
      app \
  && adduser \
      --uid 1000 \
      --ingroup app \
      --shell /bin/sh \
      --disabled-password \
      --no-create-home \
      app \
  && apk add --no-cache \
      ca-certificates=20260611-r0 \
      tzdata=2026c-r0 \
  && update-ca-certificates

WORKDIR /app

COPY --from=build --chown=app:app /out/main /usr/local/bin/geomixer

USER 1000:1000

CMD [ "geomixer" ]
STOPSIGNAL SIGTERM
