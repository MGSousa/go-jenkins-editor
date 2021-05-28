FROM golang:alpine3.13 as build

RUN apk add --no-cache curl git

WORKDIR /app

COPY . ./

SHELL ["/bin/ash", "-eo", "pipefail", "-c"]

RUN curl -sL https://git.io/goreleaser | /bin/sh -s -- release --snapshot


FROM alpine:3.13

ENV APP_ENV="production"

WORKDIR /app

COPY --from=build /app/dist/go-jenkins-editor_linux_amd64/go-jenkins-editor ./

ENTRYPOINT ["/app/go-jenkins-editor"]