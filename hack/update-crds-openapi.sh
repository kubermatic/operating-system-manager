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

CONTAINERIZE_IMAGE=quay.io/kubermatic/build:go-1.24-node-20-5 containerize ./hack/update-crds-openapi.sh
SCRIPT_ROOT=$(dirname "${BASH_SOURCE}")

echodate "Creating vendor directory"
go mod vendor

echodate "Generating OpenAPI 3 schema for CRDs"
go run sigs.k8s.io/controller-tools/cmd/controller-gen \
  crd \
  object:headerFile="hack/header.txt" \
  paths=./pkg/crd/... \
  output:crd:artifacts:config=./deploy/crd

echodate "Generating CustomOperatingSystemProfile CRD"
CUSTOM_OSP=./hack/kkp/operatingsystemmanager.k8c.io_customoperatingsystemprofiles.yaml
yq -e -i '.spec.versions = load("./deploy/crd/operatingsystemmanager.k8c.io_operatingsystemprofiles.yaml").spec.versions' $CUSTOM_OSP
