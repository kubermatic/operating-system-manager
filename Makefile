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

export GOPATH?=$(shell go env GOPATH)
export CGO_ENABLED=0
export GOPROXY?=https://proxy.golang.org
export GO111MODULE=on
export GOFLAGS?=-mod=readonly -trimpath

CMD = $(notdir $(wildcard ./cmd/*))
BUILD_DEST ?= _build

.PHONY: lint
lint:
	@golangci-lint --version
	golangci-lint run -v ./pkg/...

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
	go build -o $@ ./cmd/$*

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
