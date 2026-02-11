# Needs to be defined before including Makefile.common to auto-generate targets
DOCKER_ARCHS ?= amd64 arm64 ppc64le
DOCKER_REPO	 ?= treydock
export GOPATH ?= $(firstword $(subst :, ,$(shell go env GOPATH)))
GOLANG_CROSS_VERSION ?= v1.25.7

include Makefile.common

DOCKER_IMAGE_NAME ?= cgroup_exporter

coverage:
	go test -race -coverpkg=./... -coverprofile=coverage.txt -covermode=atomic ./...

release-test:
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
	-v `pwd`:/work -w /work \
	ghcr.io/goreleaser/goreleaser-cross:$(GOLANG_CROSS_VERSION) \
	build --snapshot --clean

release:
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
	--env-file .release-env -v `pwd`:/work -w /work \
	ghcr.io/goreleaser/goreleaser-cross:$(GOLANG_CROSS_VERSION) \
	release --clean

%/.unpacked: %.ttar
	@echo ">> extracting fixtures"
	./ttar -C $(dir $*) -x -f $*.ttar
	touch $@

update_fixtures:
	rm -vf fixtures/.unpacked
	./ttar -c -f fixtures.ttar fixtures/

.PHONY: test
test: fixtures/.unpacked common-test
