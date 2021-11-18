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

set -e

# Use a special env variable for machine-controller only
OSM_KUBECONFIG=${MC_KUBECONFIG:-$(dirname $0)/../.kubeconfig}
# If you want to use the default kubeconfig `export MC_KUBECONFIG=$KUBECONFIG`

rm -r $(dirname $0)/../_build/
make -C $(dirname $0)/.. build
$(dirname $0)/../_build/osm-controller \
  -kubeconfig=$OSM_KUBECONFIG \
  -namespace=cloud-init-settings \
  -worker-count=50 \
  -cni-version=v0.8.7 \
  -containerd-version=1.4
