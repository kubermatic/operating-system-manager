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

### This script sets up a local KKP installation in kind, deploys a
### couple of test Presets and Users and then runs the e2e tests for the
### external ccm-migration.

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

GOOS="${GOOS:-linux}"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-kubermatic}"
KIND_NODE_VERSION="${KIND_NODE_VERSION:-v1.22.2}"
KIND_PORT="${KIND_PORT-31000}"

type kind > /dev/null || fatal \
  "Kind is required to run this script, please refer to: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"

function clean_up {
  echodate "Deleting cluster ${KIND_CLUSTER_NAME}"
  kind delete cluster --name "${KIND_CLUSTER_NAME}" || true
}
appendTrap clean_up EXIT

# Only start docker daemon in CI envorinment.
if [[ ! -z "${JOB_NAME:-}" ]] && [[ ! -z "${PROW_JOB_ID:-}" ]]; then
  start_docker_daemon
fi

cat << EOF > kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: ${KIND_PORT}
    hostPort: 6443
    protocol: TCP
EOF

# setup Kind cluster
time retry 5 kind create cluster \
  --name="${KIND_CLUSTER_NAME}" \
  --image=kindest/node:"${KIND_NODE_VERSION}" \
  --config=kind-config.yaml
kind export kubeconfig --name=${KIND_CLUSTER_NAME}
echodate "Kind cluster created"

# install CRDs
retry 3 kubectl apply -f ./charts/crd
echodate "Finished installing CRDs"

# run e2e tests
echo "Running e2e tests..."
go test -v ./pkg...