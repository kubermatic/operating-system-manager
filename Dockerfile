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

ARG GO_VERSION=1.17.8
FROM golang:${GO_VERSION} AS builder
WORKDIR /go/src/k8c.io/operating-system-manager
COPY . .
RUN make all

FROM alpine:3.12

RUN apk add --no-cache ca-certificates cdrkit

COPY --from=builder \
    /go/src/k8c.io/operating-system-manager/_build/osm-controller \
    /go/src/k8c.io/operating-system-manager/_build/webhook \
    /usr/local/bin/

USER nobody
