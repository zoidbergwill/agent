#!/bin/bash
set -euo pipefail

if ! command -v gometalinter &>/dev/null ; then
  go get github.com/alecthomas/gometalinter
  gometalinter --install --vendor
fi

gometalinter \
  --exclude='error return value not checked.*(Close|Log|Print).*\(errcheck\)$' \
  --exclude='.*_test\.go:.*error return value not checked.*\(errcheck\)$' \
  --exclude='duplicate of.*_test.go.*\(dupl\)$' \
  --disable=aligncheck \
  --disable=gotype \
  --disable=gas \
  --cyclo-over=20 \
  --tests \
  --deadline=20s
