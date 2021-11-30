# Build step....
#
FROM golang:alpine AS build
RUN apk add --no-cache --update bash make git curl gcc sqlite musl-dev icu-dev

RUN export CGO_ENABLED=1
COPY . /go/searcher
WORKDIR /go/searcher
RUN go get
RUN go build

# Final image...
#
FROM alpine
RUN apk add --no-cache --update ca-certificates sqlite

RUN mkdir -p /searcher/config
RUN mkdir -p /searcher/files
RUN mkdir -p /searcher/data

COPY --from=build /go/searcher/go-searcher            /searcher
COPY --from=build /go/searcher/config/searchForm.html /searcher/config

EXPOSE 8080
WORKDIR /searcher

# NOTE: if there are problems starting swap the comments on the next two
# lines to allow you to view the searcher logfiles from the host...
#
#ENTRYPOINT ["/searcher/searcher", "-l", "/searcher/data/searcher.log"]
ENTRYPOINT ["/searcher/searcher", "-l", "/tmp/searcher.log"]
