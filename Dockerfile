FROM golang:1.20.7-alpine3.17 AS builder
RUN apk update && apk add git make gcompat curl build-base
WORKDIR /go/src/app
COPY . ./
RUN make build

FROM alpine:3.17
RUN apk --no-cache add ca-certificates gcompat
WORKDIR /
COPY --from=builder /go/src/app/cgroup_exporter .
ENTRYPOINT ["/cgroup_exporter"]
