ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:glibc
ARG ARCH="amd64"
ARG OS="linux"
COPY .build/${OS}-${ARCH}/cgroup_exporter /cgroup_exporter
EXPOSE 9306
ENTRYPOINT ["/cgroup_exporter"]
