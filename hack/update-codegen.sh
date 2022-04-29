#!/usr/bin/env bash

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

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

CONTAINERIZE_IMAGE=golang:1.18.1 containerize ./hack/update-codegen.sh
SCRIPT_ROOT=$(dirname "${BASH_SOURCE}")

echodate "Creating vendor directory"
go mod vendor
chmod +x vendor/k8s.io/code-generator/generate-groups.sh

export "GOFLAGS=-mod=vendor"

echodate "Generating osm:v1alpha1"
./vendor/k8s.io/code-generator/generate-groups.sh deepcopy \
   k8c.io/operating-system-manager/pkg/crd/client \
   k8c.io/operating-system-manager/pkg/crd \
   osm:v1alpha1 \
  --go-header-file ${SCRIPT_ROOT}/header.txt

echodate "Generating reconciling functions"
go generate ./pkg/...