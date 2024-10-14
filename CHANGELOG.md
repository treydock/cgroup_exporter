## 1.0.0 / 2024-10-14

### Changes

* [ENHANCEMENT] Update to Go 1.22.3 and update Go dependencies (#34)
* [ENHANCEMENT] Remove containerd/cgroups v1 dependency (#36)
* [ENHANCEMENT] Allow custom slurm paths via --config.paths (#31)
* [ENHANCEMENT] Support cgroup v2 (#28)

## 1.0.0-rc.4 / 2024-07-19

### Changes

* [ENHANCEMENT] Update to Go 1.22.3 and update Go dependencies (#34)
* [ENHANCEMENT] Remove containerd/cgroups v1 dependency (#36)

## 1.0.0-rc.3 / 2024-05-18

### Changes

* [ENHANCEMENT] Update to Go 1.22.3 and update Go dependencies (#34)

## 1.0.0-rc.2 / 2024-05-18

### Changes

* [ENHANCEMENT] Use goreleaser to handle releases (#33)

## 1.0.0-rc.1 / 2024-05-17

### Changes

* [ENHANCEMENT] Allow custom slurm paths via --config.paths (#31)

## 1.0.0-rc.0 / 2024-01-24

### Changes

* [ENHANCEMENT] Support cgroup v2 (#28)

## 0.9.1 / 2023-05-12

### Changes

* [BUGFIX] Avoid possible nil pointer errors (#24)

## 0.9.0 / 2023-05-06

### Changes

* [CHANGE] Trim exec path in middle (#22)
* [ENHANCEMENT] Update to Go 1.20 and update Go module dependencies (#23)

## 0.8.1 / 2022-11-15

### Changes

* [BUGFIX] Avoid null references during what appears to be race condition (#21)

## 0.8.0 / 2022-03-08

### Changes

* [ENHANCEMENT] Update Go to 1.17
* [ENHANCEMENT] Update Go module dependencies

## 0.7.0 / 2021-04-23

### Changes

* [ENHANCEMENT] Update to Go 1.16
* [ENHANCEMENT] Update Go module dependencies

## 0.6.0 / 2020-10-03

* Update to Go 1.15

## 0.5.0 / 2020-10-02

* Add cgroup_process_exec_count metric
* Switch logging to promlog
* Parallelize cgroup loads and process info collection

## 0.4.0 / 2020-10-01

* Add cgroup_cpu_info metric
* Update to Go 1.14 and update dependencies

## 0.3.0 / 2020-04-03

* Add cgroup_memory_rss_bytes and cgroup_memory_cache_bytes metrics

## 0.2.1 / 2020-03-18

* Fix Dockerfile to work on supported platforms

## 0.2.0 / 2020-02-27

### Changes

* Replace swap metrics with memsw to describe the raw values

## 0.1.0 / 2020-02-20

### Changes

* Add metric to indicate collect failures and remove success metric
* Better error handling
* Combine cgroup_userslice_info and cgroup_job_info into cgroup_info
* Rename cgroup_cpu_kernel_seconds to cgroup_cpu_system_seconds

## 0.0.1 / 2020-02-20

### Changes

* Initial Release

