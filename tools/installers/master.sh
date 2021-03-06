#!/bin/bash

# Copyright 2019 The gVisor Authors.
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

# Install runsc from the master branch.
set -e

curl -fsSL https://gvisor.dev/archive.key | sudo apt-key add -
add-apt-repository "deb https://storage.googleapis.com/gvisor/releases release main"
while true; do
  if apt-get update; then
    apt-get install -y runsc
    break
  fi
  result=$?
  # Check if apt update failed to aquire the file lock.
  if [[ $result -ne 100 ]]; then
    exit $result
  fi
done
runsc install
service docker restart

