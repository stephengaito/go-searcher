# Build step....
#
FROM golang:alpine AS build
RUN apk add --no-cache --update bash make git curl gcc sqlite musl-dev icu-dev

RUN go get github.com/tidwall/gjson
RUN go get github.com/tidwall/sjson
RUN go get github.com/bvinc/go-sqlite-lite/sqlite3
RUN go get github.com/grokify/html-strip-tags-go
RUN export CGO_ENABLED=1
COPY . /go/searcher
WORKDIR /go/searcher
RUN go build

# Final image...
#
FROM alpine
RUN apk add --no-cache --update ca-certificates sqlite

RUN mkdir -p /searcher/config
RUN mkdir -p /searcher/files
RUN mkdir -p /searcher/data

COPY --from=build /go/searcher/searcher               /searcher
COPY --from=build /go/searcher/config/searchForm.html /searcher/config

EXPOSE 8080
WORKDIR /searcher
ENTRYPOINT ["/searcher/searcher -l /tmp/searcher.log"]
