FROM golang:1.13 AS builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN make build

FROM scratch
WORKDIR /
COPY --from=builder /build/cgroup_exporter /cgroup_exporter
ENTRYPOINT ["/cgroup_exporter"]
