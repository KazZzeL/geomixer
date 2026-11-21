# syntax=docker/dockerfile:1
FROM golang:1.27.4-alpine3.24@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS build

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

FROM alpine:3.24.4@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf AS runtime

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
      ca-certificates=20261020-r0 \
      tzdata=2026f-r0 \
  && update-ca-certificates

WORKDIR /app

COPY --from=build --chown=app:app /out/main /usr/local/bin/geomixer

USER 1000:1000

CMD [ "geomixer" ]
STOPSIGNAL SIGTERM
