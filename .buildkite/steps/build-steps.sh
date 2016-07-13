#!/bin/bash

set -euo pipefail

echo "steps:"

for build in "${OS_ARCH}"; do
  cat << EOF
  - name: ":package: ${build}"
    command: ".buildkite/steps/build-binary.sh ${build}"
    artifact_paths: "pkg/*"
    plugins:
      docker-compose#e8ce6c1:
        run: agent
EOF
done
