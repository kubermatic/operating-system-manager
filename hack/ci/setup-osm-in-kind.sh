#!/usr/bin/env bash

# Copyright 2022 The Operating System Manager contributors.
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

source hack/lib.sh

if [ -z "${KIND_CLUSTER_NAME:-}" ]; then
  echodate "KIND_CLUSTER_NAME must be set by calling setup-kind-cluster.sh first."
  exit 1
fi

export MACHINE_CONTROLLER_YAML="${MACHINE_CONTROLLER_YAML:-hack/ci/testdata/machine-controller.yaml}"
export OSM_VERSION="${OSM_VERSION:-$(git rev-parse HEAD)}"

echodate "Install machine controller in kind cluster..."

kubectl apply -f $MACHINE_CONTROLLER_YAML

# Build osm binary and load the Docker images into the kind cluster
echodate "Building OSM binary for $OSM_VERSION"
TEST_NAME="Build OSM binary"

beforeGoBuild=$(nowms)
time retry 1 make build
pushElapsed osm_go_build_duration_milliseconds $beforeGoBuild

beforeDockerBuild=$(nowms)

echodate "Building OSM Docker image"
TEST_NAME="Build OSM Docker image"
IMAGE_NAME="quay.io/kubermatic/operating-system-manager:latest"
time retry 5 docker build -t "$IMAGE_NAME" .
time retry 5 kind load docker-image "$IMAGE_NAME" --name "$KIND_CLUSTER_NAME"

pushElapsed osm_docker_build_duration_milliseconds $beforeDockerBuild
echodate "Successfully built and loaded osm image"

echodate "Install osm in kind cluster..."
kubectl create namespace cloud-init-settings
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.7.1/cert-manager.yaml

# Wait for cert-manager to be ready
kubectl -n cert-manager rollout status deploy/cert-manager
kubectl -n cert-manager rollout status deploy/cert-manager-cainjector
kubectl -n cert-manager rollout status deploy/cert-manager-webhook

# Install certificate
kubectl apply -f deploy/certificate.yaml

# Install resources
kubectl apply -f deploy/crd/
kubectl apply -f deploy/
