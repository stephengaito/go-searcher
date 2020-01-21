# Build step....
#
FROM golang:alpine AS build
RUN apk add --no-cache --update bash make git curl gcc sqlite musl-dev icu-dev

RUN go get gopkg.in/yaml.v2
RUN go get github.com/grokify/html-strip-tags-go
RUN go get --tags "sqlite_fts5 sqlite_icu" github.com/mattn/go-sqlite3
RUN export CGO_ENABLED=1
COPY . /go/searcher
WORKDIR /go/searcher
RUN go build

# Final image...
#
FROM alpine
RUN apk add --no-cache --update ca-certificates sqlite

RUN mkdir /searcher
COPY --from=build /go/searcher/searcher        /searcher
COPY --from=build /go/searcher/searchForm.html /searcher

EXPOSE 8080
WORKDIR /searcher
ENTRYPOINT ["/searcher/searcher -l /tmp/searcher.log"]
