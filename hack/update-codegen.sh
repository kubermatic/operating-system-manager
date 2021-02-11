#!/bin/bash

# Copyright 2021 The Kubermatic Kubernetes Platform contributors.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

CONTAINERIZE_IMAGE=golang:1.15.1 containerize ./hack/update-codegen.sh
SCRIPT_ROOT=$(dirname "${BASH_SOURCE}")

echodate "Creating vendor directory"
go mod vendor
chmod +x vendor/k8s.io/code-generator/generate-groups.sh

echodate "Removing old clients"
rm -rf "pkg/crd/client"

echo "" > /tmp/headerfile

# -trimpath would cause the code generation to fail, so undo the
# Makefile's value and also force mod=readonly here
export "GOFLAGS=-mod=readonly"

echodate "Generating osm:v1alpha1"
./vendor/k8s.io/code-generator/generate-groups.sh all \
   k8c.io/operating-system-manager/v1alpha1/pkg/crd/client \
   k8c.io/operating-system-manager/v1alpha1/pkg/crd \
   osm:v1alpha1 \
  --go-header-file ${SCRIPT_ROOT}/header.txt

cp -r v1alpha1/* .
rm -rf v1alpha1/

rm /tmp/headerfile
