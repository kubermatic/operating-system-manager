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

### Contains commonly used functions for the other scripts.

# Required for signal propagation to work so
# the cleanup trap gets executed when a script
# receives a SIGINT
set -o monitor

echodate() {
  # do not use -Is to keep this compatible with macOS
  echo "[$(date +%Y-%m-%dT%H:%M:%S%:z)]" "$@"
}

containerize() {
  local cmd="$1"
  local image="${CONTAINERIZE_IMAGE:-quay.io/kubermatic/util:1.4.1}"
  local gocache="${CONTAINERIZE_GOCACHE:-/tmp/.gocache}"

  if ! [ -f /.dockerenv ]; then
    echodate "Running $cmd in a Docker container using $image..."

    exec docker run \
      -v $PWD:/go/src/k8c.io/operating-system-manager \
      -w /go/src/k8c.io/operating-system-manager \
      -e "GOCACHE=$gocache" \
      -u "$(id -u):$(id -g)" \
      --rm \
      -it \
      $image $cmd $@

    exit $?
  fi
}

