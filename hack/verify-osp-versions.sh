#!/usr/bin/env bash

# Copyright 2026 The Operating System Manager contributors.
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

# Verifies that every OSP manifest in deploy/osps/default carries the version
# of the git tag pointing at HEAD. Exits 0 silently on untagged commits.

set -euo pipefail

cd "$(dirname "$0")/.."

GIT_TAG="$(git tag --points-at HEAD)"

if [[ -z "${GIT_TAG}" ]]; then
  exit 0
fi

failed=0
for osp in deploy/osps/default/*.yaml; do
  version="$(grep -E '^  version: "v[0-9]+\.[0-9]+\.[0-9]+"$' "${osp}" | cut -d'"' -f2 || true)"
  if [[ "${version}" != "${GIT_TAG}" ]]; then
    echo "${osp}: expected ${GIT_TAG}, found ${version:-no version line}" >&2
    failed=1
  fi
done

exit "${failed}"
