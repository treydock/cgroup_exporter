# cgroup Prometheus exporter

[![Build Status](https://circleci.com/gh/treydock/cgroup_exporter/tree/master.svg?style=shield)](https://circleci.com/gh/treydock/cgroup_exporter)
[![GitHub release](https://img.shields.io/github/v/release/treydock/cgroup_exporter?include_prereleases&sort=semver)](https://github.com/treydock/cgroup_exporter/releases/latest)
![GitHub All Releases](https://img.shields.io/github/downloads/treydock/cgroup_exporter/total)

# Check mount Prometheus exporter

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

## Metrics

Example of metrics exposed by this exporter when looking at `/user.slice` paths:

```
cgroup_cpu_kernel_seconds{cgroup="/user.slice/user-20821.slice"} 1.96
cgroup_cpu_total_seconds{cgroup="/user.slice/user-20821.slice"} 3.817500568
cgroup_cpu_user_seconds{cgroup="/user.slice/user-20821.slice"} 1.61
cgroup_cpus{cgroup="/user.slice/user-20821.slice"} 0
cgroup_memory_fail_count{cgroup="/user.slice/user-20821.slice"} 0
cgroup_memory_total_bytes{cgroup="/user.slice/user-20821.slice"} 6.8719476736e+10
cgroup_memory_used_bytes{cgroup="/user.slice/user-20821.slice"} 6.90176e+06
cgroup_swap_fail_count{cgroup="/user.slice/user-20821.slice"} 0
cgroup_swap_total_bytes{cgroup="/user.slice/user-20821.slice"} 9.223371968135295e+18
cgroup_swap_used_bytes{cgroup="/user.slice/user-20821.slice"} 0
cgroup_userslice_info{cgroup="/user.slice/user-20821.slice",uid="20821",username="tdockendorf"} 1
```

Example of metrics exposed by this exporter when looking at `/slurm` paths:

```
cgroup_cpu_kernel_seconds{cgroup="/slurm/uid_20821/job_12"} 0
cgroup_cpu_total_seconds{cgroup="/slurm/uid_20821/job_12"} 0.007840451
cgroup_cpu_user_seconds{cgroup="/slurm/uid_20821/job_12"} 0
cgroup_cpus{cgroup="/slurm/uid_20821/job_12"} 2
cgroup_exporter_success 1
cgroup_job_info{cgroup="/slurm/uid_20821/job_12",jobid="12",uid="20821",username="tdockendorf"} 1
cgroup_memory_total_bytes{cgroup="/slurm/uid_20821/job_12"} 2.147483648e+09
cgroup_memory_used_bytes{cgroup="/slurm/uid_20821/job_12"} 315392
cgroup_swap_total_bytes{cgroup="/slurm/uid_20821/job_12"} 2.147483648e+09
cgroup_swap_used_bytes{cgroup="/slurm/uid_20821/job_12"} 315392
```

Example of metrics exposed by this exporter when looking at `/torque` paths:

```
cgroup_cpu_kernel_seconds{cgroup="/torque/1182958.pitzer-batch.ten.osc.edu"} 26.35
cgroup_cpu_total_seconds{cgroup="/torque/1182958.pitzer-batch.ten.osc.edu"} 939.568245515
cgroup_cpu_user_seconds{cgroup="/torque/1182958.pitzer-batch.ten.osc.edu"} 915.61
cgroup_cpus{cgroup="/torque/1182958.pitzer-batch.ten.osc.edu"} 40
cgroup_exporter_success 1
cgroup_job_info{cgroup="/torque/1182958.pitzer-batch.ten.osc.edu",jobid="1182958",uid="",username=""} 1
cgroup_memory_total_bytes{cgroup="/torque/1182958.pitzer-batch.ten.osc.edu"} 1.96755132416e+11
cgroup_memory_used_bytes{cgroup="/torque/1182958.pitzer-batch.ten.osc.edu"} 5.3434466304e+10
cgroup_swap_total_bytes{cgroup="/torque/1182958.pitzer-batch.ten.osc.edu"} 1.96755132416e+11
cgroup_swap_used_bytes{cgroup="/torque/1182958.pitzer-batch.ten.osc.edu"} 5.3434466304e+10
```
