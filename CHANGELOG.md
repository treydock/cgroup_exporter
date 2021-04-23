## Unreleased

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

