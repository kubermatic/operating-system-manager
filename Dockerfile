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

ARG GO_VERSION=1.27.4
# the builder always runs on the build host's platform and cross-compiles for the target platform
FROM --platform=$BUILDPLATFORM docker.io/golang:${GO_VERSION} AS builder
ARG GOPROXY
ENV GOPROXY=${GOPROXY:-https://proxy.golang.org,direct}
WORKDIR /go/src/k8c.io/operating-system-manager

# platform-independent, shared between all target platforms
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# declared as late as possible: every layer after this runs once per target platform
ARG TARGETOS TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH make all

FROM alpine:3.23

# RUN steps in this stage execute as the target platform (via QEMU for non-native platforms)
RUN apk upgrade --no-cache && apk add --no-cache ca-certificates

COPY --from=builder \
    /go/src/k8c.io/operating-system-manager/_build/osm-controller \
    /go/src/k8c.io/operating-system-manager/_build/webhook \
    /usr/local/bin/

USER nobody
