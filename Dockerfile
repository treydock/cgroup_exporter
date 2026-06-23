FROM golang:1.26.4-alpine3.24 AS builder
RUN apk update && apk add git make gcompat curl build-base
WORKDIR /go/src/app
COPY . ./
RUN make build

FROM alpine:3.24
RUN apk --no-cache add ca-certificates gcompat
WORKDIR /
COPY --from=builder /go/src/app/cgroup_exporter .
ENTRYPOINT ["/cgroup_exporter"]
