FROM golang:alpine3.13 as build

RUN apk add --no-cache curl git

WORKDIR /app

COPY . ./

SHELL ["/bin/ash", "-eo", "pipefail", "-c"]

RUN curl -sL https://git.io/goreleaser | /bin/sh -s -- release --snapshot


FROM alpine:3.13

WORKDIR /app

COPY --from=build /app/dist/go-jenkins-editor_linux_amd64/go-jenkins-editor ./
COPY --from=build /app/.env ./

ENTRYPOINT ["/app/go-jenkins-editor"]