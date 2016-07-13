#!/bin/bash

set -euo pipefail

if [[ "$BUILDKITE_BUILD_NUMBER" == "" ]]; then
  echo "Error: Missing \$BUILDKITE_BUILD_NUMBER"
  exit 1
fi

echo "--- :$1: Building $1/$2"

rm -rf pkg

./scripts/utils/build-binary.sh $1 $2 $BUILDKITE_BUILD_NUMBER
