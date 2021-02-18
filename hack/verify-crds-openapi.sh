#!/usr/bin/env bash

# Copyright 2021 The Kubermatic Kubernetes Platform contributors.
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

TEMPDIR=_tmp
DIFFROOT="charts/crd"
TMP_DIFFROOT="$TEMPDIR/charts/crd"

cleanup() {
  rm -rf "$TEMPDIR"
}
trap "cleanup" EXIT SIGINT

cleanup

mkdir -p "${TMP_DIFFROOT}"
cp -a "${DIFFROOT}"/* "${TMP_DIFFROOT}"

./hack/update-crds-openapi.sh

echodate "Diffing ${DIFFROOT} against freshly generated CRDs"
ret=0
diff -Naupr "${DIFFROOT}" "${TMP_DIFFROOT}" || ret=$?
cp -a "${TMP_DIFFROOT}"/* "${DIFFROOT}"
if [[ $ret -eq 0 ]]; then
  echodate "${DIFFROOT} up to date."
else
  echodate "${DIFFROOT} is out of date. Please run hack/update-crds-openapi.sh"
  exit 1
fi
