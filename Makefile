# Copyright 2021 The Operating System Manager contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

SHELL = /bin/bash -eu -o pipefail

export GOPATH?=$(shell go env GOPATH)
export CGO_ENABLED=0
export GOPROXY?=https://proxy.golang.org
export GO111MODULE=on
export GOFLAGS?=-mod=readonly -trimpath
export GIT_TAG ?= $(shell git tag --points-at HEAD)

GO_VERSION = 1.27.4

CMD = $(notdir $(wildcard ./cmd/*))
BUILD_DEST ?= _build

REGISTRY ?= quay.io
REGISTRY_NAMESPACE ?= kubermatic

IMAGE_TAG = \
		$(shell echo $$(git rev-parse HEAD && if [[ -n $$(git status --porcelain) ]]; then echo '-dirty'; fi)|tr -d ' ')
IMAGE_REPO ?= $(REGISTRY)/$(REGISTRY_NAMESPACE)/operating-system-manager
IMAGE_NAME ?= $(IMAGE_REPO):$(IMAGE_TAG)
PLATFORMS ?= linux/amd64,linux/arm64
# GOPROXY without a value is taken from the environment, if set
DOCKER_BUILD_ARGS = --build-arg GO_VERSION=$(GO_VERSION) --build-arg GOPROXY

BASE64_ENC = \
		$(shell if base64 -w0 <(echo "") &> /dev/null; then echo "base64 -w0"; else echo "base64 -b0"; fi)

.PHONY: lint
lint:
	@golangci-lint --version
	golangci-lint run -v

yamllint:
	yamllint -c .yamllint.conf .

.PHONY: vendor
vendor: buildenv
	go mod vendor

.PHONY: buildenv
buildenv:
	@go version

.PHONY: verify-boilerplate
verify-boilerplate:
	./hack/verify-boilerplate.sh

.PHONY: verify-licence
verify-licence: GOFLAGS = -mod=readonly
verify-licence: vendor
	wwhrd check

.PHONY: verify-codegen
verify-codegen: GOFLAGS = -mod=readonly
verify-codegen: vendor
	./hack/verify-codegen.sh

.PHONY: update-codegen
update-codegen: GOFLAGS = -mod=readonly
update-codegen: vendor
	./hack/update-codegen.sh

.PHONY: verify-crds-openapi
verify-crds-openapi: GOFLAGS = -mod=readonly
verify-crds-openapi: vendor
	./hack/verify-crds-openapi.sh

.PHONY: update-crds-openapi
update-crds-openapi: GOFLAGS = -mod=readonly
update-crds-openapi: vendor
	./hack/update-crds-openapi.sh

.PHONY: all
all: build

.PHONY: build
build: $(CMD)

.PHONY: $(CMD)
$(CMD): %: $(BUILD_DEST)/%

$(BUILD_DEST)/%: cmd/%
	go build -v -o $@ ./cmd/$*

.PHONY: run
run:
	./hack/run-operating-system-manager.sh

.PHONY: test
test:
	go test -v ./pkg...

.PHONY: clean
clean:
	rm -rf $(BUILD_DEST)
	@echo "Cleaned $(BUILD_DEST)"

.PHONY: update-fixtures
update-fixtures:
	./hack/update-fixtures.sh

.PHONY: download-gocache
download-gocache:
	@./hack/ci/download-gocache.sh

# single-arch image for the host platform, loaded into the local image store
.PHONY: docker-image
docker-image:
	docker build $(DOCKER_BUILD_ARGS) -t $(IMAGE_NAME) .

# builds the image for all $(PLATFORMS) without pushing or loading it;
# requires a docker-container buildx builder, see hack/ci/setup-docker-buildx.sh
.PHONY: docker-image-multiarch
docker-image-multiarch:
	docker buildx build $(DOCKER_BUILD_ARGS) --platform $(PLATFORMS) .

# builds the multi-arch image once and pushes it with the commit tag and,
# for tagged commits, the git tag(s) and latest; all tags share one image index,
# so the SBOMs only need to be attached once
.PHONY: docker-image-publish
docker-image-publish:
	./hack/verify-osp-versions.sh
	docker buildx build $(DOCKER_BUILD_ARGS) --platform $(PLATFORMS) \
		--tag $(IMAGE_NAME) \
		$(foreach tag,$(GIT_TAG) $(if $(GIT_TAG),latest),--tag $(IMAGE_REPO):$(tag)) \
		--push .
	./hack/attach-image-sbom.sh $(IMAGE_NAME)

.PHONY: shfmt
shfmt:
	shfmt -w -sr -i 2 hack
