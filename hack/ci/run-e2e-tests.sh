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

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

export GIT_HEAD_HASH="$(git rev-parse HEAD)"
export OSM_VERSION="${GIT_HEAD_HASH}"

TEST_NAME="Pre-warm Go build cache"
echodate "Attempting to pre-warm Go build cache"

beforeGocache=$(nowms)
make download-gocache
pushElapsed gocache_download_duration_milliseconds $beforeGocache

echodate "Creating kind cluster"
source hack/ci/setup-kind-cluster.sh

echodate "Setting up OSM in kind on revision ${OSM_VERSION}"

beforeOSMSetup=$(nowms)

source hack/ci/setup-osm-in-kind.sh
pushElapsed kind_osm_setup_duration_milliseconds $beforeOSMSetup
