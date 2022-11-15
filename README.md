# cgroup Prometheus exporter

[![Build Status](https://circleci.com/gh/treydock/cgroup_exporter/tree/master.svg?style=shield)](https://circleci.com/gh/treydock/cgroup_exporter)
[![GitHub release](https://img.shields.io/github/v/release/treydock/cgroup_exporter?include_prereleases&sort=semver)](https://github.com/treydock/cgroup_exporter/releases/latest)
![GitHub All Releases](https://img.shields.io/github/downloads/treydock/cgroup_exporter/total)
[![codecov](https://codecov.io/gh/treydock/cgroup_exporter/branch/master/graph/badge.svg)](https://codecov.io/gh/treydock/cgroup_exporter)

# cgroup Prometheus exporter

The `cgroup_exporter` produces metrics from cgroups.

This exporter by default listens on port `9306` and all metrics are exposed via the `/metrics` endpoint.

# Usage

The `--config.paths` flag is required and must point to paths of cgroups to monitor. If there is `/sys/fs/cgroup/cpuacct/user.slice` then the value for `--config.paths` would be `/user.slice`.

## Docker

Example of running the Docker container

```
docker run -d -p 9306:9306 -v "/:/host:ro,rslave" treydock/cgroup_exporter --path.cgroup.root=/host/sys/fs/cgroup
```

## Install

Download the [latest release](https://github.com/treydock/cgroup_exporter/releases)

## Build from source

To produce the `cgroup_exporter` binaries:

```
make build
```

Or

```
go get github.com/treydock/cgroup_exporter
```

## Process metrics

If you wish to collect process information for a cgroup pass the `--collect.proc` flag. If this exporter is not running as root then it's required to set capabilities to ensure the user running this exporter can read everything under procfs:

```
setcap cap_sys_ptrace=eip /usr/bin/cgroup_exporter
```

## Metrics

Example of metrics exposed by this exporter when looking at `/user.slice` paths:

```
cgroup_cpu_system_seconds{cgroup="/user.slice/user-20821.slice"} 1.96
cgroup_cpu_total_seconds{cgroup="/user.slice/user-20821.slice"} 3.817500568
cgroup_cpu_user_seconds{cgroup="/user.slice/user-20821.slice"} 1.61
cgroup_cpus{cgroup="/user.slice/user-20821.slice"} 0
cgroup_cpu_info{cgroup="/user.slice/user-20821.slice",cpus=""} 1
cgroup_info{cgroup="/user.slice/user-20821.slice",uid="20821",username="tdockendorf",jobid=""} 1
cgroup_memory_cache_bytes{cgroup="/user.slice/user-20821.slice"} 2.322432e+06
cgroup_memory_fail_count{cgroup="/user.slice/user-20821.slice"} 0
cgroup_memory_rss_bytes{cgroup="/user.slice/user-20821.slice"} 5.378048e+06
cgroup_memory_total_bytes{cgroup="/user.slice/user-20821.slice"} 6.8719476736e+10
cgroup_memory_used_bytes{cgroup="/user.slice/user-20821.slice"} 6.90176e+06
cgroup_memsw_fail_count{cgroup="/user.slice/user-20821.slice"} 0
cgroup_memsw_total_bytes{cgroup="/user.slice/user-20821.slice"} 9.223371968135295e+18
cgroup_memsw_used_bytes{cgroup="/user.slice/user-20821.slice"} 0
```

Example of metrics exposed by this exporter when looking at `/slurm` paths:

```
cgroup_cpu_system_seconds{cgroup="/slurm/uid_20821/job_12"} 0
cgroup_cpu_total_seconds{cgroup="/slurm/uid_20821/job_12"} 0.007840451
cgroup_cpu_user_seconds{cgroup="/slurm/uid_20821/job_12"} 0
cgroup_cpus{cgroup="/slurm/uid_20821/job_12"} 2
cgroup_cpu_info{cgroup="/slurm/uid_20821/job_12",cpus="0,1"} 1
cgroup_info{cgroup="/slurm/uid_20821/job_12",jobid="12",uid="20821",username="tdockendorf"} 1
cgroup_memory_cache_bytes{cgroup="/slurm/uid_20821/job_12"} 4.096e+03
cgroup_memory_fail_count{cgroup="/slurm/uid_20821/job_12"} 0
cgroup_memory_rss_bytes{cgroup="/slurm/uid_20821/job_12"} 3.11296e+05
cgroup_memory_total_bytes{cgroup="/slurm/uid_20821/job_12"} 2.147483648e+09
cgroup_memory_used_bytes{cgroup="/slurm/uid_20821/job_12"} 315392
cgroup_memsw_fail_count{cgroup="/slurm/uid_20821/job_12"} 0
cgroup_memsw_total_bytes{cgroup="/slurm/uid_20821/job_12"} 2.147483648e+09
cgroup_memsw_used_bytes{cgroup="/slurm/uid_20821/job_12"} 315392
```

Example of metrics exposed by this exporter when looking at `/torque` paths:

```
cgroup_cpu_system_seconds{cgroup="/torque/1182958.batch.example.com"} 26.35
cgroup_cpu_total_seconds{cgroup="/torque/1182958.batch.example.com"} 939.568245515
cgroup_cpu_user_seconds{cgroup="/torque/1182958.batch.example.com"} 915.61
cgroup_cpus{cgroup="/torque/1182958.batch.example.com"} 8
cgroup_cpu_info{cgroup="/torque/1182958.batch.example.com",cpus="0,1,2,3,4,5,6,7,8"} 1
cgroup_info{cgroup="/torque/1182958.batch.example.com",jobid="1182958",uid="",username=""} 1
cgroup_memory_cache_bytes{cgroup="/torque/1182958.batch.example.com"} 1.09678592e+08
cgroup_memory_fail_count{cgroup="/torque/1182958.batch.example.com"} 0
cgroup_memory_rss_bytes{cgroup="/torque/1182958.batch.example.com"} 8.2444320768e+10
cgroup_memory_total_bytes{cgroup="/torque/1182958.batch.example.com"} 1.96755132416e+11
cgroup_memory_used_bytes{cgroup="/torque/1182958.batch.example.com"} 5.3434466304e+10
cgroup_memsw_fail_count{cgroup="/torque/1182958.batch.example.com"} 0
cgroup_memsw_total_bytes{cgroup="/torque/1182958.batch.example.com"} 1.96755132416e+11
cgroup_memsw_used_bytes{cgroup="/torque/1182958.batch.example.com"} 5.3434466304e+10
```
