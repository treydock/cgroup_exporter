FROM golang:1.22.3-alpine3.19 AS builder
RUN apk update && apk add git make gcompat curl build-base
WORKDIR /go/src/app
COPY . ./
RUN make build

FROM alpine:3.19
RUN apk --no-cache add ca-certificates gcompat
WORKDIR /
COPY --from=builder /go/src/app/cgroup_exporter .
ENTRYPOINT ["/cgroup_exporter"]
