# Build step....
#
FROM golang:alpine AS build
RUN apk add --no-cache --update bash make git curl gcc sqlite-dev musl-dev icu-dev

COPY . /searcher
WORKDIR /searcher/searcher
RUN go get --tags "icu fts5" && go build --tags "icu fts5"

# Final image...
#
FROM alpine
RUN apk add --no-cache --update sqlite-libs icu-libs icu-data

RUN mkdir -p /searcher/config /searcher/files /searcher/data

COPY --from=build /searcher/searcher/searcher /searcher

EXPOSE 8080
WORKDIR /searcher

ENTRYPOINT ["/searcher/searcher", "-c", "/searcher/config/searcher.jsonc"]
