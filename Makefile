# Needs to be defined before including Makefile.common to auto-generate targets
DOCKER_ARCHS ?= amd64 armv7 arm64 ppc64le
DOCKER_REPO	 ?= treydock

include Makefile.common

DOCKER_IMAGE_NAME ?= cgroup_exporter

coverage:
	go test -race -coverpkg=./... -coverprofile=coverage.txt -covermode=atomic ./...

%/.unpacked: %.ttar
	@echo ">> extracting fixtures"
	./ttar -C $(dir $*) -x -f $*.ttar
	touch $@

update_fixtures:
	rm -vf fixtures/.unpacked
	./ttar -c -f fixtures.ttar fixtures/

.PHONY: test
test: fixtures/.unpacked common-test
